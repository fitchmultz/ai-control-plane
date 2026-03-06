#!/usr/bin/env python3
"""
Mock LLM Upstream Server for Offline Demo Mode

Purpose: Provide a lightweight, OpenAI-compatible upstream server for the AI
Control Plane offline demo. This supports LiteLLM integration without external
network access or provider API keys by returning deterministic mock responses.

Responsibilities:
- Implement the minimal OpenAI-compatible endpoints used by offline mode:
  - `GET /health`
  - `GET /v1/models` and `GET /v1/models/{id}`
  - `POST /v1/chat/completions` (including `stream=true` SSE-style streaming)
- Produce deterministic mock assistant content using `MOCK_RESPONSE_TEMPLATE`.
- Emit OpenAI-compatible response envelopes including `usage` fields suitable for
  LiteLLM budget/rate-limit demos.
- Support latency/token-estimation tuning via environment variables.

Non-scope:
- Does NOT perform real model inference or call external providers.
- Does NOT implement authentication/authorization, key validation, or TLS.
- Does NOT implement the full OpenAI API surface (embeddings, images, files, etc.).
- Does NOT provide exact provider tokenization; token counts are heuristic only.
- Does NOT persist conversation state; each request is handled independently.

Invariants/Assumptions:
- `AVAILABLE_MODELS` IDs must stay in sync with `demo/config/litellm-offline.yaml`
  (currently: `mock-gpt`, `mock-claude`).
- Token counting is a simple heuristic: `max(1, int(len(text) * MOCK_TOKENS_PER_CHAR))`.
- The `{user_message}` placeholder is truncated to 100 characters before formatting.
- Expected runtime is Docker Compose offline mode where the service is reachable at
  `http://mock-upstream:8080` (internal network); it is demo-only.
"""

import os
import time
import json
import uuid
import logging
from datetime import datetime
from functools import wraps
from flask import Flask, request, jsonify, Response

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('mock-upstream')

app = Flask(__name__)

# Configuration from environment variables
MOCK_LATENCY_MS = int(os.environ.get('MOCK_LATENCY_MS', '100'))
MOCK_TOKENS_PER_CHAR = float(os.environ.get('MOCK_TOKENS_PER_CHAR', '0.25'))
MOCK_RESPONSE_TEMPLATE = os.environ.get(
    'MOCK_RESPONSE_TEMPLATE',
    'This is a mock response from the {model} offline demo model. '
    'Your message was: "{user_message}"'
)

# Available mock models (must match litellm-offline.yaml)
AVAILABLE_MODELS = [
    {
        "id": "mock-gpt",
        "object": "model",
        "created": 1677610602,
        "owned_by": "mock-provider",
        "permission": [],
        "root": "mock-gpt",
        "parent": None,
    },
    {
        "id": "mock-claude",
        "object": "model",
        "created": 1677610602,
        "owned_by": "mock-provider",
        "permission": [],
        "root": "mock-claude",
        "parent": None,
    },
]


def simulate_latency():
    """Simulate network latency to upstream provider."""
    if MOCK_LATENCY_MS > 0:
        time.sleep(MOCK_LATENCY_MS / 1000.0)


def estimate_tokens(text: str) -> int:
    """
    Estimate token count from text.
    Uses a simple character-based heuristic (~4 chars per token).
    """
    if not text:
        return 0
    return max(1, int(len(text) * MOCK_TOKENS_PER_CHAR))


def generate_mock_response(user_message: str, model: str) -> str:
    """Generate a deterministic mock response based on user input."""
    # Use template with user message excerpt
    excerpt = user_message[:100] + "..." if len(user_message) > 100 else user_message
    return MOCK_RESPONSE_TEMPLATE.format(model=model, user_message=excerpt)


def create_chat_completion_response(
    request_id: str,
    model: str,
    messages: list,
    response_content: str,
    stream: bool = False
) -> dict:
    """Create an OpenAI-compatible chat completion response."""
    # Calculate prompt tokens from all messages
    prompt_text = "\n".join(
        f"{msg.get('role', 'user')}: {msg.get('content', '')}"
        for msg in messages
    )
    prompt_tokens = estimate_tokens(prompt_text)
    completion_tokens = estimate_tokens(response_content)
    total_tokens = prompt_tokens + completion_tokens

    response = {
        "id": f"chatcmpl-mock-{request_id}",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": model,
        "choices": [
            {
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": response_content,
                },
                "finish_reason": "stop",
            }
        ],
        "usage": {
            "prompt_tokens": prompt_tokens,
            "completion_tokens": completion_tokens,
            "total_tokens": total_tokens,
        },
    }

    return response


def create_streaming_response(
    request_id: str,
    model: str,
    messages: list,
    response_content: str
):
    """Create an OpenAI-compatible streaming response."""
    # Calculate tokens
    prompt_text = "\n".join(
        f"{msg.get('role', 'user')}: {msg.get('content', '')}"
        for msg in messages
    )
    prompt_tokens = estimate_tokens(prompt_text)
    completion_tokens = estimate_tokens(response_content)
    current_time = int(time.time())

    # Split response into chunks for streaming simulation
    words = response_content.split()
    chunk_size = max(1, len(words) // 5)  # ~5 chunks

    # First chunk: role
    role_chunk = {
        "id": f"chatcmpl-mock-{request_id}",
        "object": "chat.completion.chunk",
        "created": current_time,
        "model": model,
        "choices": [{"index": 0, "delta": {"role": "assistant"}, "finish_reason": None}]
    }
    yield f"data: {json.dumps(role_chunk)}\n\n"

    # Content chunks
    for i in range(0, len(words), chunk_size):
        chunk_words = words[i:i + chunk_size]
        chunk_content = " ".join(chunk_words)
        if i + chunk_size < len(words):
            chunk_content += " "

        content_chunk = {
            "id": f"chatcmpl-mock-{request_id}",
            "object": "chat.completion.chunk",
            "created": current_time,
            "model": model,
            "choices": [{"index": 0, "delta": {"content": chunk_content}, "finish_reason": None}]
        }
        yield f"data: {json.dumps(content_chunk)}\n\n"

        # Small delay between chunks for realistic streaming
        time.sleep(0.01)

    # Final chunk with usage (OpenAI doesn't typically include usage in streaming,
    # but LiteLLM may expect it for budget tracking)
    final_chunk = {
        "id": f"chatcmpl-mock-{request_id}",
        "object": "chat.completion.chunk",
        "created": current_time,
        "model": model,
        "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}],
        "usage": {
            "prompt_tokens": prompt_tokens,
            "completion_tokens": completion_tokens,
            "total_tokens": prompt_tokens + completion_tokens
        }
    }
    yield f"data: {json.dumps(final_chunk)}\n\n"

    yield "data: [DONE]\n\n"


@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint for Docker and load balancers."""
    return jsonify({
        "status": "healthy",
        "service": "mock-upstream",
        "version": "1.0.0",
        "models_available": len(AVAILABLE_MODELS)
    })


@app.route('/v1/models', methods=['GET'])
def list_models():
    """List available mock models (OpenAI-compatible)."""
    simulate_latency()

    return jsonify({
        "object": "list",
        "data": AVAILABLE_MODELS,
    })


@app.route('/v1/chat/completions', methods=['POST'])
def chat_completions():
    """
    Handle chat completion requests (OpenAI-compatible).

    Returns deterministic mock responses with accurate token counts
    for budget and rate limit testing.
    """
    simulate_latency()

    try:
        data = request.get_json()
        if not data:
            return jsonify({"error": "Invalid JSON body"}), 400

        model = data.get('model', 'mock-gpt')
        messages = data.get('messages', [])
        stream = data.get('stream', False)

        if not messages:
            return jsonify({"error": "No messages provided"}), 400

        # Get the last user message for context-aware response
        user_message = ""
        for msg in reversed(messages):
            if msg.get('role') == 'user':
                user_message = msg.get('content', '')
                break

        # Generate response
        response_content = generate_mock_response(user_message, model)

        # Create unique request ID
        request_id = str(uuid.uuid4())[:8]

        logger.info(f"Mock request: model={model}, stream={stream}, "
                    f"prompt_chars={len(user_message)}, "
                    f"response_chars={len(response_content)}")

        if stream:
            # Return streaming response
            return Response(
                create_streaming_response(request_id, model, messages, response_content),
                mimetype='text/plain',
                headers={
                    'Cache-Control': 'no-cache',
                    'Content-Type': 'text/event-stream',
                }
            )
        else:
            # Return standard response
            response = create_chat_completion_response(
                request_id, model, messages, response_content, stream=False
            )
            return jsonify(response)

    except Exception as e:
        logger.error(f"Error processing request: {e}")
        return jsonify({
            "error": {
                "message": str(e),
                "type": "internal_error",
            }
        }), 500


@app.route('/v1/models/<path:model_id>', methods=['GET'])
def get_model(model_id):
    """Get a specific model by ID."""
    simulate_latency()

    for model in AVAILABLE_MODELS:
        if model['id'] == model_id:
            return jsonify(model)

    return jsonify({"error": f"Model '{model_id}' not found"}), 404


@app.errorhandler(404)
def not_found(error):
    """Handle 404 errors gracefully."""
    return jsonify({
        "error": {
            "message": "Endpoint not found",
            "type": "not_found_error",
        }
    }), 404


@app.errorhandler(500)
def internal_error(error):
    """Handle 500 errors gracefully."""
    logger.error(f"Internal server error: {error}")
    return jsonify({
        "error": {
            "message": "Internal server error",
            "type": "internal_error",
        }
    }), 500


if __name__ == '__main__':
    port = int(os.environ.get('PORT', '8080'))
    debug = os.environ.get('DEBUG', 'false').lower() == 'true'

    logger.info(f"Starting Mock Upstream Server on port {port}")
    logger.info(f"Configuration: latency={MOCK_LATENCY_MS}ms, "
                f"tokens_per_char={MOCK_TOKENS_PER_CHAR}")
    logger.info(f"Available models: {[m['id'] for m in AVAILABLE_MODELS]}")

    # Run the Flask app
    app.run(host='0.0.0.0', port=port, debug=debug)
