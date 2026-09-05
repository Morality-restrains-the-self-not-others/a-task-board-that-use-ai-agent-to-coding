# ============================================================
# Claude Agent Docker Image (Go version)
# ============================================================
# Bundles claude-agent (Go static binary) + Claude Code (Node.js CLI)
# into a single container — ~60% smaller than the Python version.
#
# Single runtime: Node.js + Go binary (no Python, no pip, no venv)
#
# Build modes:
#   Multi-stage (default): docker build -t claude-agent .
#   Pre-built binary:      ./build.sh && docker build --build-arg USE_PREBUILT=1 -t claude-agent .
#
# Run:
#   docker run --rm -e ANTHROPIC_API_KEY claude-agent run "your task"
#   docker run --rm -it -e ANTHROPIC_API_KEY claude-agent interactive
# ============================================================

ARG USE_PREBUILT=0

# Stage 1: Build the Go binary (only when USE_PREBUILT != 1)
FROM golang:1.22-alpine AS go-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download 2>/dev/null || true

COPY main.go ./
COPY src/ ./src/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /claude-agent .

# Stage 2: Runtime
FROM node:20-slim

LABEL org.opencontainers.image.title="claude-agent"
LABEL org.opencontainers.image.description="Claude Code wrapper (Go sidecar) with config, trajectory, and Docker support"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/anthropics/claude-code"

# Install git (needed by Claude Code for repository operations)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code (npm)
COPY package.json /opt/claude-agent/package.json
WORKDIR /opt/claude-agent
RUN npm install --omit=dev && \
    ln -s /opt/claude-agent/node_modules/.bin/claude /usr/local/bin/claude

# Copy Go binary — multi-stage build (default) or pre-built
COPY --from=go-builder /claude-agent /usr/local/bin/claude-agent
# Override with pre-built binary when USE_PREBUILT=1:
#   Comment out the line above and uncomment below:
#   COPY bin/claude-agent /usr/local/bin/claude-agent

# Pre-create trajectory directory
RUN mkdir -p /workspace/.trajectories && chmod 777 /workspace

WORKDIR /workspace
VOLUME /workspace

ENTRYPOINT ["claude-agent"]
CMD ["--help"]
