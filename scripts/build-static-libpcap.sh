#!/usr/bin/env bash
#
# Build a static libpcap into PREFIX so a cgo niac binary needs libc alone.
#
#   scripts/build-static-libpcap.sh PREFIX [configure args...]
#
# Link with CGO_CFLAGS=-IPREFIX/include and CGO_LDFLAGS=-LPREFIX/lib. The
# directory holds only libpcap.a, so gopacket's -lpcap resolves to the archive
# and the binary carries no libpcap SONAME.
#
# The shared E2E workflow runs the daemon inside the Playwright image, which
# has no libpcap and cannot apt-install one. Built from source because Debian's
# libpcap.a is built with RDMA, D-Bus, libnl, and systemd support, and a static
# link would drag those libraries in. niac opens captures with pcap_open_live
# only. Needs a C compiler, make, flex, and bison.
set -euo pipefail

LIBPCAP_VERSION=1.10.7
LIBPCAP_SHA256=0b394ac90dbc0a9838ff97468e05c9c9a3e873dec2514cd58db65d859d296e31

if [ $# -lt 1 ]; then
  echo "usage: $0 PREFIX [configure args...]" >&2
  exit 2
fi
prefix=$1
shift

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$work"

curl -fsSLO "https://www.tcpdump.org/release/libpcap-${LIBPCAP_VERSION}.tar.gz"
echo "${LIBPCAP_SHA256}  libpcap-${LIBPCAP_VERSION}.tar.gz" | sha256sum -c -
tar xzf "libpcap-${LIBPCAP_VERSION}.tar.gz"
cd "libpcap-${LIBPCAP_VERSION}"

./configure --prefix="$prefix" --disable-shared --disable-rdma --disable-dbus \
  --without-libnl --disable-bluetooth --disable-usb --disable-netmap "$@"
make -j"$(nproc)"
make install

if [ ! -f "$prefix/lib/libpcap.a" ]; then
  echo "::error::no static libpcap in $prefix" >&2
  exit 1
fi
