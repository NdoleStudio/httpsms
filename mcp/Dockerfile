# syntax=docker/dockerfile:1

# ---- builder -----------------------------------------------------------
# Compiles a static (CGO_ENABLED=0) binary from ./cmd/server. Building in a
# throwaway stage keeps the Go toolchain, module cache, and source tree out
# of the final image.
FROM golang:1.25-alpine AS builder

ARG GIT_COMMIT

WORKDIR /src

# Copy go.mod/go.sum first so `go mod download` is cached across builds that
# only change application source, not dependencies.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w -X main.Version=${GIT_COMMIT}" \
    -o /out/mcp-server ./cmd/server

# ---- runtime -------------------------------------------------------------
# alpine (not scratch) so ca-certificates and tzdata are trivially
# installable: this service makes outbound HTTPS calls to the httpSMS API,
# Firebase's JWKS endpoint, and OAuth client metadata documents, all of
# which require a validated CA trust store.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S mcp && adduser -S mcp -G mcp

# Run as the unprivileged "mcp" user for the entire runtime stage: Cloud Run
# does not require this, but nothing in this service needs root.
USER mcp

COPY --from=builder /out/mcp-server /usr/local/bin/mcp-server

# Cloud Run injects PORT at runtime and the server binds to it
# (see mcp/internal/config.Config.Port, default 8080); EXPOSE documents the
# default for local `docker run` and `docker image inspect`.
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/mcp-server"]
