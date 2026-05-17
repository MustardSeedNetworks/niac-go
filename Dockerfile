# =============================================================================
# NiAC container image — multi-stage, libpcap-aware
# =============================================================================
# Cloud Native Buildpacks can't build this image because the default builders
# don't include libpcap-dev (gopacket/pcap CGO dependency). This Dockerfile
# explicitly installs libpcap-dev in the build stage and ships only the runtime
# library + binary in the final stage.
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: build the embedded React/Vite UI
# -----------------------------------------------------------------------------
FROM node:25-bookworm AS ui-build
WORKDIR /src/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build
# Vite outputs to ../internal/api/ui (via outDir in vite.config.ts); copy that
# tree out so the next stage can mount it via COPY --from.
RUN mkdir -p /out && cp -r ../internal/api/ui /out/ui

# -----------------------------------------------------------------------------
# Stage 2: build the Go binary with CGO + libpcap
# -----------------------------------------------------------------------------
FROM golang:1.25-bookworm AS go-build
RUN apt-get update \
    && apt-get install -y --no-install-recommends libpcap-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Place the prebuilt UI where //go:embed expects it, then build with CGO.
COPY --from=ui-build /out/ui ./internal/api/ui
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN UI_HASH=$(find internal/api/ui -type f -exec md5sum {} \; | sort | md5sum | cut -d' ' -f1) \
    && CGO_ENABLED=1 go build -trimpath -buildvcs=false \
        -ldflags="-s -w -X github.com/krisarmstrong/niac-go/internal/version.Version=${VERSION} -X github.com/krisarmstrong/niac-go/internal/version.Commit=${COMMIT} -X github.com/krisarmstrong/niac-go/internal/version.BuildTime=${BUILD_DATE} -X github.com/krisarmstrong/niac-go/internal/version.UIBuildHash=${UI_HASH}" \
        -o /out/niac ./cmd/niac

# -----------------------------------------------------------------------------
# Stage 3: minimal runtime
# -----------------------------------------------------------------------------
FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libpcap0.8 \
        ca-certificates \
        libcap2-bin \
        tini \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system niac \
    && useradd --system --gid niac --home-dir /var/lib/niac --shell /usr/sbin/nologin niac \
    && mkdir -p /etc/niac /var/lib/niac /var/log/niac \
    && chown -R niac:niac /etc/niac /var/lib/niac /var/log/niac \
    && chmod 0750 /etc/niac /var/lib/niac /var/log/niac

COPY --from=go-build /out/niac /usr/bin/niac
# Raw-socket capability so the daemon can run as the unprivileged niac user.
RUN setcap 'cap_net_raw,cap_net_admin=+ep' /usr/bin/niac

USER niac
WORKDIR /var/lib/niac
EXPOSE 8080

# OCI labels for traceability.
ARG VERSION=dev
ARG COMMIT=unknown
LABEL org.opencontainers.image.title="niac" \
      org.opencontainers.image.source="https://github.com/krisarmstrong/niac-go" \
      org.opencontainers.image.licenses="Proprietary" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}"

# tini reaps zombies and forwards signals so SIGTERM cleanly shuts down the daemon.
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/bin/niac"]
CMD ["daemon", "--listen", ":8080"]
