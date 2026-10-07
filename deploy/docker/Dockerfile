# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e
#
# NIAC from a published release, as a container. The image installs the
# release's Linux archive, checked against that release's checksums.txt, so
# it carries the same binary as the .deb and .rpm (libpcap linked in, glibc
# only) rather than a second build of the source.
#
#   docker build --build-arg NIAC_VERSION=0.111.0 -t niac:0.111.0 deploy/docker
#   docker run -d --name niac --network host \
#     --cap-add NET_RAW --cap-add NET_ADMIN \
#     -v niac-data:/var/lib/niac niac:0.111.0
#
# docs/DEPLOYMENT.md explains the host network and the two capabilities.

FROM debian:trixie-slim@sha256:a99cfc517144bc59b1978475ec53b46ecabec7e43635402ee5b77cc54cd1b20a AS fetch

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
ARG NIAC_VERSION
ARG TARGETARCH
RUN test -n "${NIAC_VERSION}" || { echo 'set --build-arg NIAC_VERSION=X.Y.Z' >&2; exit 1; }
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl libcap2-bin \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /fetch
RUN archive="niac-${NIAC_VERSION#v}-linux-${TARGETARCH}.tar.gz" \
    && base="https://github.com/MustardSeedNetworks/niac-go/releases/download/v${NIAC_VERSION#v}" \
    && curl -fsSLO "${base}/${archive}" \
    && curl -fsSLO "${base}/checksums.txt" \
    && grep -E "  ${archive}\$" checksums.txt | sha256sum -c - \
    && tar -xzf "${archive}" niac \
    && setcap 'cap_net_raw,cap_net_admin=+ep' niac

FROM debian:trixie-slim@sha256:a99cfc517144bc59b1978475ec53b46ecabec7e43635402ee5b77cc54cd1b20a

# The daemon runs unprivileged, as the packages run it. The file capabilities
# set in the fetch stage travel with COPY; they let that user open raw sockets,
# and take effect only when the container is granted NET_RAW and NET_ADMIN and
# does not set no-new-privileges. A fixed uid lets an operator chown a bind
# mount to match.
COPY --from=fetch /fetch/niac /usr/bin/niac
RUN groupadd --system --gid 10001 niac \
    && useradd --system --uid 10001 --gid niac --home-dir /var/lib/niac --shell /usr/sbin/nologin niac \
    && install -d -o niac -g niac -m 0750 /var/lib/niac

# The same layout the systemd unit gives the service: every piece of state —
# configs, recovery records, tokens, the self-signed certificate — lives
# under /var/lib/niac, so one volume carries it across image upgrades.
ENV HOME=/var/lib/niac \
    NIAC_CERT_DIR=/var/lib/niac/certs
USER 10001:10001
WORKDIR /var/lib/niac
VOLUME ["/var/lib/niac"]
EXPOSE 8445/tcp
ENTRYPOINT ["/usr/bin/niac"]
CMD ["daemon"]
