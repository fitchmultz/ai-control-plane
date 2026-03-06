# Mock LLM Upstream Server for Offline Demo Mode
#
# This container provides an OpenAI-compatible mock LLM service for
# demonstrating AI Control Plane governance features without requiring
# actual provider API keys.
#
# Usage:
#   docker build -t mock-upstream .
#   docker run -p 8080:8080 -e MOCK_LATENCY_MS=100 mock-upstream

FROM python:3.14-alpine@sha256:faee120f7885a06fcc9677922331391fa690d911c020abb9e8025ff3d908e510

# Set working directory
WORKDIR /app
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_ROOT_USER_ACTION=ignore

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip==26.0.1 \
    && pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY mock_server.py .

# Create non-root user for security
# Create non-root user (adduser for Alpine compatibility)
RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

# Expose the service port
EXPOSE 8080

# Environment variables with defaults
ENV PORT=8080 \
    MOCK_LATENCY_MS=100 \
    MOCK_TOKENS_PER_CHAR=0.25 \
    PYTHONUNBUFFERED=1

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/health')" || exit 1

# Run with gunicorn for production-like behavior
# Using 2 workers for demo purposes (adjust based on load)
CMD ["gunicorn", "--bind", "0.0.0.0:8080", "--workers", "2", "--worker-class", "sync", "--access-logfile", "-", "--error-logfile", "-", "mock_server:app"]
