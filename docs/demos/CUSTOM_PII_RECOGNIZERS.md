# Custom PII Recognizers Guide

## Overview

The AI Control Plane supports custom PII recognizers to detect organization-specific sensitive data patterns that are not covered by Presidio's built-in entities.

This guide covers the **advanced Presidio path**. Use it when native LiteLLM guardrails are not sufficient for your entity-detection requirements (for example internal account formats, employee IDs, or proprietary key patterns).

**Important:** When using `RECOGNIZER_REGISTRY_CONF_FILE`, the registry file becomes **authoritative**. You must include both predefined (built-in) and custom recognizers in the same file. Custom-only configurations will unintentionally drop built-in recognizers.

## Quick Start

Custom recognizers are defined in `demo/config/presidio/recognizers/custom_recognizers.yaml` and automatically loaded by the Presidio Analyzer on startup via the `RECOGNIZER_REGISTRY_CONF_FILE` environment variable.

## Registry File Schema

The recognizer registry file requires these top-level keys:

```yaml
# Required: Languages supported by this registry
supported_languages:
  - en

# Required: Global regex flags (Python re module)
# 26 = IGNORECASE(2) + MULTILINE(8) + DOTALL(16)
global_regex_flags: 26

# Required: List of all recognizers (predefined + custom)
recognizers:
  # Predefined (built-in) recognizers
  - name: "CreditCardRecognizer"
    type: predefined
  
  # Custom recognizers
  - name: "My Custom Recognizer"
    type: custom
    supported_entity: "MY_ENTITY"
    # ... pattern definition
```

## Pattern Recognizer Schema

Each custom recognizer requires:

| Field | Description | Required |
|-------|-------------|----------|
| `name` | Human-readable recognizer name | Yes |
| `type` | Either `predefined` or `custom` | Yes |
| `supported_entity` | Entity type identifier (used in litellm.yaml) | Yes (custom only) |
| `supported_language` | Language code (e.g., "en") | Yes (custom only) |
| `patterns` | List of regex patterns with scores | Yes (custom only) |
| `context` | Context words that boost confidence | No |

### Pattern Definition

```yaml
patterns:
  - name: "pattern_identifier"
    regex: "\\bPATTERN-HERE\\b"
    score: 0.85  # Confidence score (0.0-1.0)
```

## Required Predefined Recognizers

When using a custom registry file, you must explicitly list predefined recognizers you want to keep:

```yaml
recognizers:
  # Core PII recognizers
  - name: "UsSsnRecognizer"
    type: predefined
  - name: "UsPassportRecognizer"
    type: predefined
  - name: "UsLicenseRecognizer"
    type: predefined
  - name: "UsBankRecognizer"
    type: predefined
  - name: "UsItinRecognizer"
    type: predefined
  - name: "CreditCardRecognizer"
    type: predefined
  - name: "CryptoRecognizer"
    type: predefined
  - name: "IbanRecognizer"
    type: predefined
  - name: "EmailRecognizer"
    type: predefined
  - name: "PhoneRecognizer"
    type: predefined
  
  # NLP-based recognizer (for PERSON, LOCATION)
  - name: "SpacyRecognizer"
    type: predefined
```

## Adding a New Custom Recognizer

### Step 1: Define the Pattern

Identify the data format and create a regex pattern:

```yaml
- name: "My Custom Recognizer"
  type: custom
  supported_entity: "MY_CUSTOM_ENTITY"
  supported_language: en
  patterns:
    - name: "my_pattern"
      regex: "\\bMYCODE-[0-9]{6}\\b"
      score: 0.85
  context:
    - code
    - id
```

### Step 2: Add to LiteLLM Configuration

Add the entity action to `demo/config/litellm.yaml`:

```yaml
pii_entities_config:
  MY_CUSTOM_ENTITY: "BLOCK"  # or "MASK"
```

**Important:** The `supported_entity` value in the recognizer must match exactly the key in `pii_entities_config`.

### Step 3: Restart Services

```bash
make down
make up
```

### Step 4: Verify Detection

```bash
# Test with curl
curl -X POST http://localhost:4000/v1/chat/completions \
  -H "Authorization: Bearer $DEMO_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-haiku-4-5", "messages": [{"role": "user", "content": "My code is MYCODE-123456"}]}'
```

## Included Example Recognizers

| Entity | Format | Action | Purpose |
|--------|--------|--------|---------|
| AWS_ACCESS_KEY | AKIA[16 chars] | BLOCK | AWS credentials |
| ACP_EMPLOYEE_ID | EMP-XXXXXX or ACPEMP-XXXXXXXX | BLOCK | Employee identification |
| ACP_CUSTOMER_ACCOUNT | CUST-XXXXXXXX or AC-XXXXXXXXXX | BLOCK | Customer accounts |
| ACP_SYSTEM_KEY | ACPKEY-XXXX-XXXX-XXXX | BLOCK | Internal credentials |
| ACP_PROJECT_CODE | PROJ-XXXX-XXXX | MASK | Project references |

## Testing

Run Scenario 6 with custom PII tests:

```bash
make demo-scenario SCENARIO=6
```

## Troubleshooting

### Custom Entities Not Detected

1. Verify the YAML syntax is valid
2. Check the container logs: `docker compose logs presidio-analyzer`
3. Ensure the entity name matches exactly between recognizer and litellm.yaml
4. Verify predefined recognizers are included in the registry file

### Regex Not Matching

1. Test the regex pattern in a regex tester
2. Use word boundaries (`\b`) to avoid partial matches
3. Adjust the score threshold if needed
4. Check that `global_regex_flags` is set correctly

### Built-in Entities Missing

When using `RECONIZER_REGISTRY_CONF_FILE`, the file replaces the default registry. You must explicitly include predefined recognizers you want to keep.

## Reference

- [Presidio Custom Recognizers](https://microsoft.github.io/presidio/analyzer/adding_recognizers/)
- [Presidio Pattern Recognizer](https://microsoft.github.io/presidio/analyzer/developing_recognizers/)
