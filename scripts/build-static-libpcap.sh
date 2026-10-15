#!/usr/bin/env bash
#
# Build a static libpcap into PREFIX, for a cgo seed binary that needs libc
# alone at run time.
#
#   scripts/build-static-libpcap.sh PREFIX [configure args...]
#
# Extra arguments go to libpcap's configure, e.g. a cross build:
#   scripts/build-static-libpcap.sh /opt/libpcap-static/arm64 \
#     --host=aarch64-linux-gnu CC=aarch64-linux-gnu-gcc
#
# Link against it with CGO_CFLAGS=-IPREFIX/include, and either name the
# archive (CGO_LDFLAGS=PREFIX/lib/libpcap.a, as .goreleaser.yml does) or put
# PREFIX/lib first on the search path (CGO_LDFLAGS=-LPREFIX/lib): it holds only
# libpcap.a, so gopacket's `-lpcap` resolves to the archive. Either way the
# binary carries no libpcap SONAME.
#
# Callers:
#   * release.yml: the one Linux binary is packaged as both .deb and .rpm.
#     Debian names its libpcap `libpcap.so.0.8` and Fedora `libpcap.so.1`, so a
#     dynamic link crash-looped seed.service on every Fedora install (#2978).
#   * ci.yml e2e-backend: the E2E daemon runs inside the Playwright image,
#     which has no libpcap and cannot apt-install one (#2986).
#
# Built from source because Debian's libpcap.a is built with RDMA, D-Bus,
# libnl and systemd support, and a static link drags every one of them in.
# seed opens captures with pcap_open_live only, so none of those backends is
# used. Needs a C compiler, make, flex and bison.
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
