// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Contain the production-only deployment validation helpers used by the
//     profile-aware config validation entrypoint.
//
// Responsibilities:
//   - Validate canonical production secrets, compose, and Caddy inputs.
//   - Enforce database and ingress runtime requirements for production.
//   - Provide stable helper seams for production validation unit tests.
//
// Scope:
//   - Production-specific validation logic only.
//
// Usage:
//   - Called by ValidateDeploymentConfig when the production profile is active.
//
// Invariants/Assumptions:
//   - Production secrets default to the canonical host-side secrets path.
//   - Remote OTEL ingress must terminate through authenticated TLS ingress.
package validation

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/policy"
	"gopkg.in/yaml.v3"
)

func validateProductionDeploymentConfig(repoRoot string, opts ConfigValidationOptions) ([]string, error) {
	secretsEnvFile := strings.TrimSpace(opts.SecretsEnvFile)
	if secretsEnvFile == "" {
		secretsEnvFile = defaultProductionSecretsEnvFile
	}

	issues := make([]string, 0)
	secretsValues, secretsIssues, err := validateProductionSecretsEnvFile(secretsEnvFile)
	if err != nil {
		return nil, err
	}
	issues = append(issues, secretsIssues...)
	issues = append(issues, validateCanonicalProductionCompose(repoRoot)...)
	issues = append(issues, validateCanonicalProductionCaddyfile(repoRoot)...)
	if len(secretsValues) > 0 {
		issues = append(issues, validateProductionRuntimeValues(secretsEnvFile, secretsValues)...)
	}
	return issues, nil
}

func validateProductionSecretsEnvFile(path string) (map[string]string, []string, error) {
	issues := make([]string, 0)
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []string{fmt.Sprintf("%s: missing canonical production secrets file", path)}, nil
		}
		return nil, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		issues = append(issues, fmt.Sprintf("%s: secrets file must not be a symlink", path))
		return nil, issues, nil
	}
	if !info.Mode().IsRegular() {
		issues = append(issues, fmt.Sprintf("%s: secrets file must be a regular file", path))
		return nil, issues, nil
	}
	if info.Mode().Perm()&0o077 != 0 {
		issues = append(issues, fmt.Sprintf("%s: secrets file permissions must deny group/other access (found %04o)", path, info.Mode().Perm()))
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return nil, nil, err
	}
	if !dirInfo.IsDir() {
		issues = append(issues, fmt.Sprintf("%s: parent path must be a directory", filepath.Dir(path)))
	} else if dirInfo.Mode().Perm()&0o022 != 0 {
		issues = append(issues, fmt.Sprintf("%s: parent directory permissions must not be group/other writable (found %04o)", filepath.Dir(path), dirInfo.Mode().Perm()))
	}

	keys := []string{
		"ACP_DATABASE_MODE",
		"CADDY_ACME_CA",
		"CADDY_DOMAIN",
		"CADDY_EMAIL",
		"CADDYFILE_PATH",
		"DATABASE_URL",
		"LITELLM_MASTER_KEY",
		"LITELLM_PUBLISH_HOST",
		"LITELLM_PUBLIC_URL",
		"LITELLM_SALT_KEY",
		"OTEL_INGEST_AUTH_TOKEN",
		"OTEL_PUBLISH_HOST",
		"POSTGRES_DB",
		"POSTGRES_PASSWORD",
		"POSTGRES_USER",
		"CADDY_PUBLISH_HOST",
	}

	values := make(map[string]string, len(keys))
	secretsFile := config.NewEnvFile(path)
	for _, key := range keys {
		value, ok, lookupErr := secretsFile.Lookup(key)
		if lookupErr != nil {
			return nil, nil, lookupErr
		}
		if ok {
			values[key] = strings.TrimSpace(value)
		}
	}

	required := []string{"LITELLM_MASTER_KEY", "LITELLM_SALT_KEY", "DATABASE_URL"}
	for _, key := range required {
		if strings.TrimSpace(values[key]) == "" {
			issues = append(issues, fmt.Sprintf("%s: required key %s is missing or empty", path, key))
		}
	}

	return values, issues, nil
}

func validateCanonicalProductionCompose(repoRoot string) []string {
	target := policy.SurfaceTarget{Path: "demo/docker-compose.yml"}
	root, err := policy.LoadYAMLTarget(repoRoot, target)
	if err != nil {
		return []string{fmt.Sprintf("%s: invalid YAML: %v", target.Path, err)}
	}

	serviceNode := serviceMapping(root, "otel-collector")
	if serviceNode == nil {
		return []string{fmt.Sprintf("%s: service %q must exist", target.Path, "otel-collector")}
	}
	portsNode := policy.MappingValue(serviceNode, "ports")
	if portsNode == nil || portsNode.Kind != yaml.SequenceNode {
		return []string{fmt.Sprintf("%s: service %q must define ports", target.Path, "otel-collector")}
	}

	issues := make([]string, 0)
	expectedPorts := map[string]string{
		"4317":  "127.0.0.1",
		"4318":  "127.0.0.1",
		"13133": "127.0.0.1",
	}
	seenPorts := make(map[string]struct{}, len(expectedPorts))
	for _, portNode := range portsNode.Content {
		portSpec := strings.TrimSpace(policy.ScalarValue(portNode))
		if portSpec == "" {
			continue
		}
		if strings.Contains(portSpec, "OTEL_PUBLISH_HOST") {
			issues = append(issues, fmt.Sprintf("%s: service %q must not use OTEL_PUBLISH_HOST; raw OTEL ports are localhost-only", target.Path, "otel-collector"))
		}
		host, targetPort := splitComposePort(portSpec)
		expectedHost, tracked := expectedPorts[targetPort]
		if !tracked {
			continue
		}
		seenPorts[targetPort] = struct{}{}
		if host != expectedHost {
			issues = append(issues, fmt.Sprintf("%s: service %q port %s must bind %s (found %s)", target.Path, "otel-collector", targetPort, expectedHost, host))
		}
	}
	for targetPort, expectedHost := range expectedPorts {
		if _, ok := seenPorts[targetPort]; !ok {
			issues = append(issues, fmt.Sprintf("%s: service %q must publish %s on %s", target.Path, "otel-collector", targetPort, expectedHost))
		}
	}
	return issues
}

func validateCanonicalProductionCaddyfile(repoRoot string) []string {
	path := filepath.Join(repoRoot, filepath.FromSlash("demo/config/caddy/Caddyfile.prod"))
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("demo/config/caddy/Caddyfile.prod: %v", err)}
	}
	content := string(data)
	issues := make([]string, 0)
	requiredSnippets := []string{
		"handle_path /otel/*",
		`@authorized header Authorization "Bearer {$OTEL_INGEST_AUTH_TOKEN}"`,
		"reverse_proxy @authorized otel-collector:4318",
		`respond "OTEL ingest authorization required" 401`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			issues = append(issues, fmt.Sprintf("demo/config/caddy/Caddyfile.prod: missing required production OTEL ingress contract %q", snippet))
		}
	}
	return issues
}

func validateProductionRuntimeValues(path string, values map[string]string) []string {
	issues := make([]string, 0)
	issues = append(issues, validateSecretValue(path, values, "LITELLM_MASTER_KEY", 32)...)
	issues = append(issues, validateSecretValue(path, values, "LITELLM_SALT_KEY", 32)...)

	if rawOTELHost := strings.TrimSpace(values["OTEL_PUBLISH_HOST"]); rawOTELHost != "" {
		issues = append(issues, fmt.Sprintf("%s: OTEL_PUBLISH_HOST is not allowed in the production contract; raw OTEL ports are fixed to localhost and remote ingest must use TLS /otel/*", path))
	}
	if litellmHost := normalizeHostValue(values["LITELLM_PUBLISH_HOST"], "127.0.0.1"); litellmHost != "127.0.0.1" && litellmHost != "localhost" {
		issues = append(issues, fmt.Sprintf("%s: LITELLM_PUBLISH_HOST must remain localhost in production; expose traffic through Caddy TLS instead", path))
	}

	mode := normalizeDatabaseModeValue(values["ACP_DATABASE_MODE"])
	if mode == "" {
		issues = append(issues, fmt.Sprintf("%s: ACP_DATABASE_MODE must be explicitly set to embedded or external in production", path))
	}

	databaseURL := strings.TrimSpace(values["DATABASE_URL"])
	parsedDatabaseURL, err := url.Parse(databaseURL)
	if err != nil || parsedDatabaseURL == nil {
		issues = append(issues, fmt.Sprintf("%s: DATABASE_URL must be a valid PostgreSQL connection string", path))
	} else {
		issues = append(issues, validateDatabaseURL(path, values, mode, parsedDatabaseURL)...)
	}

	caddyHost := normalizeHostValue(values["CADDY_PUBLISH_HOST"], "127.0.0.1")
	if caddyHost != "127.0.0.1" && caddyHost != "localhost" {
		issues = append(issues, validateExternalTLSExposure(path, values)...)
	}

	return issues
}

func validateDatabaseURL(path string, values map[string]string, mode string, parsed *url.URL) []string {
	issues := make([]string, 0)
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "postgresql" && scheme != "postgres" {
		issues = append(issues, fmt.Sprintf("%s: DATABASE_URL must use postgres/postgresql scheme", path))
	}

	host := normalizeURLHost(parsed.Hostname())
	if host == "" {
		issues = append(issues, fmt.Sprintf("%s: DATABASE_URL must include a database host", path))
		return issues
	}

	if mode == "embedded" {
		if host != "postgres" {
			issues = append(issues, fmt.Sprintf("%s: embedded ACP_DATABASE_MODE requires DATABASE_URL host postgres (found %s)", path, host))
		}
		user := strings.TrimSpace(values["POSTGRES_USER"])
		password := strings.TrimSpace(values["POSTGRES_PASSWORD"])
		dbName := strings.TrimSpace(values["POSTGRES_DB"])
		if user == "" || password == "" || dbName == "" {
			issues = append(issues, fmt.Sprintf("%s: embedded ACP_DATABASE_MODE requires POSTGRES_USER, POSTGRES_PASSWORD, and POSTGRES_DB", path))
			return issues
		}
		issues = append(issues, validatePostgresPassword(path, password)...)
		if parsed.User == nil || parsed.User.Username() != user {
			issues = append(issues, fmt.Sprintf("%s: DATABASE_URL username must match POSTGRES_USER", path))
		}
		if parsedPassword, ok := parsed.User.Password(); !ok || parsedPassword != password {
			issues = append(issues, fmt.Sprintf("%s: DATABASE_URL password must match POSTGRES_PASSWORD", path))
		}
		databaseName := strings.Trim(strings.TrimSpace(parsed.Path), "/")
		if databaseName != dbName {
			issues = append(issues, fmt.Sprintf("%s: DATABASE_URL database name must match POSTGRES_DB", path))
		}
		return issues
	}

	if host == "postgres" {
		issues = append(issues, fmt.Sprintf("%s: external ACP_DATABASE_MODE must not target the embedded postgres host", path))
	}
	if !isLoopbackHost(host) {
		sslMode := strings.ToLower(strings.TrimSpace(parsed.Query().Get("sslmode")))
		if sslMode != "require" && sslMode != "verify-ca" && sslMode != "verify-full" {
			issues = append(issues, fmt.Sprintf("%s: external DATABASE_URL must set sslmode=require or stronger for non-local hosts", path))
		}
	}
	return issues
}

func validateExternalTLSExposure(path string, values map[string]string) []string {
	issues := make([]string, 0)
	if strings.TrimSpace(values["CADDYFILE_PATH"]) != canonicalProductionCaddyfile {
		issues = append(issues, fmt.Sprintf("%s: exposed production ingress must use CADDYFILE_PATH=%s", path, canonicalProductionCaddyfile))
	}
	if strings.TrimSpace(values["CADDY_ACME_CA"]) != "letsencrypt" {
		issues = append(issues, fmt.Sprintf("%s: exposed production ingress must use CADDY_ACME_CA=letsencrypt", path))
	}

	domain := strings.TrimSpace(values["CADDY_DOMAIN"])
	email := strings.TrimSpace(values["CADDY_EMAIL"])
	if domain == "" {
		issues = append(issues, fmt.Sprintf("%s: CADDY_DOMAIN is required when CADDY_PUBLISH_HOST is exposed", path))
	}
	if email == "" {
		issues = append(issues, fmt.Sprintf("%s: CADDY_EMAIL is required when CADDY_PUBLISH_HOST is exposed", path))
	}

	publicURL := strings.TrimSpace(values["LITELLM_PUBLIC_URL"])
	parsedPublicURL, err := url.Parse(publicURL)
	if err != nil || parsedPublicURL == nil || !strings.EqualFold(parsedPublicURL.Scheme, "https") {
		issues = append(issues, fmt.Sprintf("%s: LITELLM_PUBLIC_URL must be a valid https:// URL when CADDY_PUBLISH_HOST is exposed", path))
	} else if domain != "" && !strings.EqualFold(parsedPublicURL.Hostname(), domain) {
		issues = append(issues, fmt.Sprintf("%s: LITELLM_PUBLIC_URL host must match CADDY_DOMAIN", path))
	}

	issues = append(issues, validateSecretValue(path, values, "OTEL_INGEST_AUTH_TOKEN", 32)...)
	return issues
}

func validateSecretValue(path string, values map[string]string, key string, minLength int) []string {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return []string{fmt.Sprintf("%s: required key %s is missing or empty", path, key)}
	}

	lowerValue := strings.ToLower(value)
	issues := make([]string, 0)
	if len(value) < minLength {
		issues = append(issues, fmt.Sprintf("%s: %s must be at least %d characters", path, key, minLength))
	}
	if strings.ContainsAny(value, " \t\r\n") {
		issues = append(issues, fmt.Sprintf("%s: %s must not contain whitespace", path, key))
	}
	placeholderFragments := []string{"change-me", "replace-with", "example", "placeholder", "your-"}
	for _, fragment := range placeholderFragments {
		if strings.Contains(lowerValue, fragment) {
			issues = append(issues, fmt.Sprintf("%s: %s must not use placeholder/demo values", path, key))
			break
		}
	}
	return issues
}

func validatePostgresPassword(path string, password string) []string {
	issues := make([]string, 0)
	if len(password) < 16 {
		issues = append(issues, fmt.Sprintf("%s: POSTGRES_PASSWORD must be at least 16 characters in production", path))
	}
	if strings.ContainsAny(password, " \t\r\n") {
		issues = append(issues, fmt.Sprintf("%s: POSTGRES_PASSWORD must not contain whitespace", path))
	}
	if slices.Contains([]string{"litellm", "postgres", "password", "change-me"}, strings.ToLower(strings.TrimSpace(password))) {
		issues = append(issues, fmt.Sprintf("%s: POSTGRES_PASSWORD must not use demo/default values", path))
	}
	return issues
}

func serviceMapping(root *yaml.Node, serviceName string) *yaml.Node {
	servicesNode := policy.MappingValue(root, "services")
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return nil
	}
	return policy.MappingValue(servicesNode, serviceName)
}

func splitComposePort(portSpec string) (string, string) {
	spec := strings.Trim(portSpec, "\"'")
	spec = strings.TrimSpace(spec)
	if strings.Contains(spec, "/") {
		spec = strings.SplitN(spec, "/", 2)[0]
	}
	if strings.HasPrefix(spec, "${") {
		closeIndex := strings.Index(spec, "}")
		if closeIndex == -1 || closeIndex+2 > len(spec) {
			return "", ""
		}
		host := spec[:closeIndex+1]
		rest := spec[closeIndex+2:]
		lastColon := strings.LastIndex(rest, ":")
		if lastColon == -1 {
			return normalizeURLHost(host), ""
		}
		return normalizeURLHost(host), strings.TrimSpace(rest[lastColon+1:])
	}
	firstColon := strings.Index(spec, ":")
	lastColon := strings.LastIndex(spec, ":")
	if firstColon == -1 || lastColon == -1 || lastColon == firstColon {
		return "", ""
	}
	return normalizeURLHost(spec[:firstColon]), strings.TrimSpace(spec[lastColon+1:])
}

func normalizeHostValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return normalizeURLHost(trimmed)
}

func normalizeURLHost(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(strings.TrimSuffix(trimmed, "]"), "["), " ")
}

func normalizeDatabaseModeValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "embedded", "external":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func isLoopbackHost(host string) bool {
	normalized := normalizeURLHost(host)
	if normalized == "localhost" {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}
