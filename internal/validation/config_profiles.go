// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Enforce profile-aware deployment configuration contracts, including the
//     production host-secrets and ingress security requirements.
//
// Responsibilities:
//   - Validate canonical production env/secrets inputs without shelling out.
//   - Enforce TLS exposure, OTEL ingress, and localhost-only raw OTEL rules.
//   - Verify canonical production config files keep the expected security shape.
//
// Scope:
//   - Read-only validation of deployment profiles and host-side env files.
//
// Usage:
//   - Called by `acpctl validate config` and Make production validation targets.
//
// Invariants/Assumptions:
//   - `/etc/ai-control-plane/secrets.env` is the canonical production secrets file.
//   - Remote OTEL ingestion must terminate at the TLS Caddy `/otel/*` ingress.
package validation

const (
	defaultProductionSecretsEnvFile = "/etc/ai-control-plane/secrets.env"
	canonicalProductionCaddyfile    = "./config/caddy/Caddyfile.prod"
)

// ConfigValidationProfile identifies the effective validation contract.
type ConfigValidationProfile string

const (
	ConfigValidationProfileDemo       ConfigValidationProfile = "demo"
	ConfigValidationProfileProduction ConfigValidationProfile = "production"
)

// ConfigValidationOptions controls profile-aware config validation.
type ConfigValidationOptions struct {
	Profile        ConfigValidationProfile
	SecretsEnvFile string
}

// ValidateDeploymentConfig validates deployment config for the requested profile.
func ValidateDeploymentConfig(repoRoot string, opts ConfigValidationOptions) ([]string, error) {
	profile := opts.Profile
	if profile == "" {
		profile = ConfigValidationProfileDemo
	}

	contractIssues, err := ValidateConfigContract(repoRoot)
	if err != nil {
		return nil, err
	}
	issues, err := ValidateDeploymentSurfaces(repoRoot)
	if err != nil {
		return nil, err
	}
	if profile != ConfigValidationProfileProduction {
		acc := NewIssues(len(contractIssues) + len(issues))
		acc.Extend(contractIssues)
		acc.Extend(issues)
		return acc.Sorted(), nil
	}

	productionIssues, err := validateProductionDeploymentConfig(repoRoot, opts)
	if err != nil {
		return nil, err
	}
	acc := NewIssues(len(contractIssues) + len(issues) + len(productionIssues))
	acc.Extend(contractIssues)
	acc.Extend(issues)
	acc.Extend(productionIssues)
	return acc.Sorted(), nil
}
