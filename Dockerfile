# Multi-stage build using Red Hat Hardened Images (Project Hummingbird).
# Builder: toolchain + package managers (not shipped in the final image).
# Runtime: minimal distroless core-runtime — no shell, no package manager.
#
# Docs: https://docs.redhat.com/en/documentation/red_hat_hardened_images/
#
# On Mac Silicon → OpenShift (amd64 workers):
#   podman build --platform=linux/amd64 -t quay.io/<user>/project-onboarding-operator:tag .

ARG HI_GO_BUILDER_IMAGE=registry.access.redhat.com/hi/go:latest-builder
ARG HI_CORE_RUNTIME_IMAGE=registry.access.redhat.com/hi/core-runtime:latest

# --- Build stage ---
FROM ${HI_GO_BUILDER_IMAGE} AS builder
ARG TARGETOS
ARG TARGETARCH

ARG VERSION=dev
ARG GIT_COMMIT=unknown

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-w -s -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" \
    -a -o /tmp/manager ./cmd

# --- Runtime stage ---
FROM ${HI_CORE_RUNTIME_IMAGE}

COPY --from=builder --chown=65532:65532 /tmp/manager /manager

USER 65532

ENTRYPOINT ["/manager"]
