# Mock LLM Upstream Server

A lightweight OpenAI-compatible mock LLM service for offline AI Control Plane demos.

## Purpose

This mock server enables AI Control Plane demonstrations without requiring:
- External network access
- OpenAI/Anthropic API keys
- Paid provider usage

It provides deterministic, reproducible responses while maintaining full compatibility
with the LiteLLM proxy for testing budgets, rate limits, and detection rules.

## Quick Start

```bash
# Build the Docker image
docker build -t mock-upstream .

# Run the mock server
docker run -p 8080:8080 mock-upstream

# Test the health endpoint
curl http://localhost:8080/health

# Test chat completions
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-gpt",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check for Docker/load balancers |
| `/v1/models` | GET | List available mock models |
| `/v1/models/{id}` | GET | Get specific model details |
| `/v1/chat/completions` | POST | Generate chat completions |

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `MOCK_LATENCY_MS` | 100 | Simulated response latency in milliseconds |
| `MOCK_TOKENS_PER_CHAR` | 0.25 | Token estimation ratio (~4 chars per token) |
| `MOCK_RESPONSE_TEMPLATE` | See below | Template for mock responses |

### Response Template

The `MOCK_RESPONSE_TEMPLATE` supports these placeholders:
- `{model}` - The model name requested
- `{user_message}` - The user's input message (truncated to 100 chars)

Default template:
```
This is a mock response from the {model} offline demo model. Your message was: "{user_message}"
```

## Available Models

- `mock-gpt` - Mock OpenAI GPT-style model
- `mock-claude` - Mock Anthropic Claude-style model

Both models return similar responses; the distinction is for testing model-specific routing and policies.

## Response Format

The mock server returns OpenAI-compatible responses:

```json
{
  "id": "chatcmpl-mock-abc123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "mock-gpt",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "This is a mock response..."
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

## Token Estimation

Token counts are estimated using a character-based heuristic (~4 characters per token).
This provides sufficient accuracy for:
- Budget tracking and enforcement
- Rate limiting (TPM - tokens per minute)
- Usage analytics

Note: This is not 100% accurate compared to actual provider tokenizers, but is sufficient for demo purposes.

## Streaming Support

The mock server supports streaming responses (`stream: true`):

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-gpt",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

## Important Notes

**This mock server is for demo and testing purposes only.**

- Responses are deterministic and do not demonstrate actual AI capabilities
- Token counts are estimated, not computed with real tokenizers
- No context awareness or conversation memory
- Production deployments must use real providers

## Docker Compose Integration

This mock server is designed to work with the AI Control Plane's offline mode:

```yaml
services:
  mock-upstream:
    build: ./mock_upstream
    environment:
      - MOCK_LATENCY_MS=100
    ports:
      - "8080:8080"
```

See `demo/docker-compose.offline.yml` for complete integration.

## Dependency Pinning

This project uses deterministic dependency management to ensure reproducible builds.

### Files

- **`requirements.in`** — Human-editable direct dependencies (Flask, gunicorn)
- **`requirements.txt`** — Machine-generated lockfile with exact versions (used by Docker)

### Updating Dependencies

To add or update a dependency:

1. Edit `requirements.in` to add/update the package
2. Regenerate the lockfile using pip-compile:

```bash
docker run --rm \
  -v "$PWD/demo/mock_upstream:/work" \
  -w /work \
  python:3.12-slim \
  sh -lc '
    pip install --quiet pip-tools
    pip-compile --no-emit-index-url --no-emit-trusted-host \
      --output-file requirements.txt requirements.in
  '
```

3. Verify the Docker build still works:

```bash
docker build -t mock-upstream demo/mock_upstream
```

### Regression Guard

A static test enforces that `requirements.txt` contains only exact version pins (`==`).
This prevents non-deterministic builds from version ranges (`>=`, `~=`, etc.).

Run the guard:
```bash
make script-tests
```
