// Package health provides health check functionality for AI Control Plane services.
//
// Purpose:
//
//	Provide comprehensive health checking for all AI Control Plane services
//	including Docker containers, LiteLLM gateway, PostgreSQL database,
//	and optional OTEL collector.
//
// Responsibilities:
//   - Check Docker container status
//   - Verify LiteLLM gateway endpoints
//   - Check PostgreSQL connectivity and schema readiness
//   - Check OTEL collector status
//   - Aggregate health results with proper exit codes
//
// Non-scope:
//   - Does not start or restart services
//   - Does not perform end-to-end LLM inference tests
//
// Invariants/Assumptions:
//   - All checks are non-invasive (read-only operations)
//   - Exit codes follow the repository contract (0/1/2/64)
package health

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

// Status represents the health status of a component
type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusUnhealthy
	StatusWarning
)

// Component represents a health check component
type Component struct {
	Name       string
	Status     Status
	Message    string
	Details    string
	IsOptional bool
}

// IsPass returns true if the component health check passed
func (c *Component) IsPass() bool {
	return c.Status == StatusHealthy || (c.IsOptional && c.Status == StatusUnknown)
}

// Result contains the overall health check result
type Result struct {
	Components []Component
	Overall    Status
}

// Checker performs health checks
type Checker struct {
	out     *output.Output
	compose *docker.Compose
	gateway *gateway.Client
	db      *db.Client
	verbose bool
}

// NewChecker creates a new health checker
func NewChecker(compose *docker.Compose, verbose bool) *Checker {
	return &Checker{
		out:     output.New(),
		compose: compose,
		gateway: gateway.NewClient(),
		db:      db.NewClient(compose),
		verbose: verbose,
	}
}

// Run performs all health checks and returns the result
func (c *Checker) Run(ctx context.Context) *Result {
	result := &Result{
		Components: make([]Component, 0),
		Overall:    StatusHealthy,
	}

	// Check 1: Docker Container Status
	c.out.SectionHeader("=== Docker Container Status ===")
	component := c.checkContainers(ctx)
	result.Components = append(result.Components, component)
	if !component.IsPass() {
		result.Overall = StatusUnhealthy
	}

	// Check 2: LiteLLM Gateway Health
	c.out.SectionHeader("=== LiteLLM Gateway Health Endpoint ===")
	component = c.checkGatewayHealth(ctx)
	result.Components = append(result.Components, component)
	if !component.IsPass() {
		result.Overall = StatusUnhealthy
	}

	// Check 3: LiteLLM Models Endpoint
	c.out.SectionHeader("=== LiteLLM Models Endpoint ===")
	component = c.checkGatewayModels(ctx)
	result.Components = append(result.Components, component)
	if component.Status == StatusUnhealthy {
		result.Overall = StatusUnhealthy
	}

	// Check 4: PostgreSQL Connectivity
	c.out.SectionHeader("=== PostgreSQL Connectivity ===")
	component = c.checkDatabase(ctx)
	result.Components = append(result.Components, component)
	if !component.IsPass() {
		result.Overall = StatusUnhealthy
	}

	// Check 5: OTEL Collector (optional)
	c.out.SectionHeader("=== OTEL Collector Status ===")
	component = c.checkOTEL(ctx)
	result.Components = append(result.Components, component)
	// OTEL is optional, so warnings don't affect overall status

	return result
}

// checkContainers checks Docker container status
func (c *Checker) checkContainers(ctx context.Context) Component {
	comp := Component{Name: "Docker Containers"}

	if c.compose == nil {
		comp.Status = StatusUnhealthy
		comp.Message = "Docker Compose not available"
		c.out.Error("Docker Compose not available")
		return comp
	}

	ps, err := c.compose.PS(ctx)
	if err != nil {
		comp.Status = StatusUnhealthy
		comp.Message = fmt.Sprintf("Failed to get container status: %v", err)
		c.out.Error(comp.Message)
		return comp
	}

	if ps == "" {
		comp.Status = StatusUnhealthy
		comp.Message = "No containers running"
		c.out.Error("No containers running")
		c.out.InfoLine("Run 'make up' to start services")
		return comp
	}

	// Check LiteLLM container
	if c.compose.IsServiceRunning(ctx, "litellm") {
		c.out.Success("LiteLLM container is running")
	} else {
		comp.Status = StatusUnhealthy
		comp.Message = "LiteLLM container is not running"
		c.out.Error(comp.Message)
	}

	// Check PostgreSQL container (only in embedded mode)
	if c.db.IsEmbedded() {
		if c.compose.IsServiceRunning(ctx, "postgres") {
			c.out.Success("PostgreSQL container is running")
		} else {
			comp.Status = StatusUnhealthy
			comp.Message = "PostgreSQL container is not running"
			c.out.Error(comp.Message)
		}
	} else {
		c.out.InfoLine("PostgreSQL container check skipped (external database mode)")
	}

	if comp.Status == StatusUnknown {
		comp.Status = StatusHealthy
		comp.Message = "All required containers are running"
	}

	return comp
}

// checkGatewayHealth checks the LiteLLM health endpoint
func (c *Checker) checkGatewayHealth(ctx context.Context) Component {
	comp := Component{Name: "LiteLLM Health Endpoint"}
	if !c.gateway.HasMasterKey() {
		comp.Status = StatusUnhealthy
		comp.Message = "LITELLM_MASTER_KEY is required for authorized health checks"
		c.out.Error(comp.Message)
		c.out.InfoLine("Set LITELLM_MASTER_KEY in demo/.env and retry")
		return comp
	}

	healthy, code, err := c.gateway.Health(ctx)
	if err != nil {
		comp.Status = StatusUnhealthy
		comp.Message = fmt.Sprintf("Authorized health endpoint check failed: %v", err)
		c.out.Error(comp.Message)
		c.out.InfoLine("Check LiteLLM logs: make logs")
		return comp
	}

	if healthy {
		comp.Status = StatusHealthy
		comp.Message = fmt.Sprintf("Health endpoint accessible (HTTP %d)", code)
		c.out.Success(comp.Message)
		if c.verbose {
			c.out.InfoLine(fmt.Sprintf("Response code: %d", code))
		}
	} else {
		comp.Status = StatusUnhealthy
		comp.Message = fmt.Sprintf("Unauthorized or unexpected HTTP status: %d", code)
		c.out.Error(comp.Message)
	}

	return comp
}

// checkGatewayModels checks the LiteLLM models endpoint
func (c *Checker) checkGatewayModels(ctx context.Context) Component {
	comp := Component{Name: "LiteLLM Models Endpoint"}
	if !c.gateway.HasMasterKey() {
		comp.Status = StatusUnhealthy
		comp.Message = "LITELLM_MASTER_KEY is required for authorized models checks"
		c.out.Error(comp.Message)
		c.out.InfoLine("Set LITELLM_MASTER_KEY in demo/.env and retry")
		return comp
	}

	accessible, code, err := c.gateway.Models(ctx)
	if err != nil {
		comp.Status = StatusUnhealthy
		comp.Message = fmt.Sprintf("Authorized models endpoint check failed: %v", err)
		c.out.Error(comp.Message)
		return comp
	}

	if accessible {
		comp.Status = StatusHealthy
		comp.Message = fmt.Sprintf("Models endpoint accessible (HTTP %d)", code)
		c.out.Success(comp.Message)
	} else {
		comp.Status = StatusUnhealthy
		comp.Message = fmt.Sprintf("Unauthorized or unexpected HTTP status: %d", code)
		c.out.Error(comp.Message)
	}

	return comp
}

// checkDatabase checks PostgreSQL connectivity
func (c *Checker) checkDatabase(ctx context.Context) Component {
	comp := Component{Name: "PostgreSQL Database"}

	if c.db.IsExternal() {
		c.out.InfoLine("External database mode detected")
		c.out.InfoLine("Database connectivity should be validated separately")
	}

	if !c.db.IsAccessible(ctx) {
		comp.Status = StatusUnhealthy
		comp.Message = "PostgreSQL is not accepting connections"
		c.out.Error(comp.Message)
		if c.db.IsEmbedded() {
			c.out.InfoLine("Check PostgreSQL logs: docker-compose logs postgres")
		}
		return comp
	}

	c.out.Success("PostgreSQL is accepting connections")

	// Check schema readiness
	if c.db.TablesExist(ctx) {
		c.out.Success("Database schema is ready (tables exist)")
		comp.Message = "Database accessible and schema ready"
	} else {
		comp.Status = StatusUnhealthy
		comp.Message = "Database schema not ready (tables missing)"
		c.out.Error(comp.Message)
	}

	if c.verbose {
		version, err := c.db.Version(ctx)
		if err == nil {
			c.out.InfoLine(fmt.Sprintf("PostgreSQL version: %s", version))
		}
	}

	if comp.Status == StatusUnknown {
		comp.Status = StatusHealthy
	}

	return comp
}

// checkOTEL checks the OTEL collector status
func (c *Checker) checkOTEL(ctx context.Context) Component {
	comp := Component{
		Name:       "OTEL Collector",
		IsOptional: true,
	}

	if c.compose == nil {
		comp.Status = StatusUnknown
		comp.Message = "Cannot check OTEL status (Docker unavailable)"
		c.out.InfoLine(comp.Message)
		return comp
	}

	if !c.compose.IsServiceRunning(ctx, "otel-collector") {
		comp.Status = StatusWarning
		comp.Message = "OTEL collector not running (optional service)"
		c.out.Warn(comp.Message)
		c.out.InfoLine("To start OTEL collector: make otel-up")
		return comp
	}

	// OTEL is running - check health endpoint
	// For now, simplified check
	c.out.Success("OTEL collector is running")
	comp.Status = StatusHealthy
	comp.Message = "OTEL collector running and healthy"

	return comp
}

// PrintSummary prints the health check summary
func (c *Checker) PrintSummary(result *Result) {
	c.out.SectionHeader("=== Health Check Summary ===")

	switch result.Overall {
	case StatusHealthy:
		c.out.Println(c.out.Green(c.out.Bold("Health check: PASSED")))
		c.out.Println("")
		c.out.Println("All required services are healthy and ready for use.")
	case StatusUnhealthy:
		c.out.Println(c.out.Red(c.out.Bold("Health check: FAILED")))
		c.out.Println("")
		c.out.Println("Some services are not healthy. See details above.")
		c.out.Println("")
		c.out.Println("Common fixes:")
		c.out.Println("  - Start services: make up")
		c.out.Println("  - View logs: make logs")
		c.out.Println("  - Restart services: make down && make up")
	default:
		c.out.Println(c.out.Yellow(c.out.Bold("Health check: UNKNOWN")))
	}
}
