#!/bin/sh
# Print the libsoup each installed Playwright WebKit build loads, and fail if
# any of them is 3.6.5.
#
# TEMPORARY (UI-SEED-28, seed#2930): Playwright 1.63's WebKit bundles libsoup
# 3.6.5, which has a use-after-free (microsoft/playwright#42803) that loses the
# session mid-run on WebKit only. The WebKit E2E jobs run on Ubuntu 26.04 so the
# build loads the OS's 3.6.6 instead; this check proves it did. Delete this
# script with the WebKit job split when Playwright 1.64 lands.
#
# The WebKit build is the one ui/'s @playwright/test resolves, not whatever
# sits in the browser cache. The network process is the one that links
# libsoup; it is resolved with the same LD_LIBRARY_PATH the MiniBrowser wrapper
# sets, so a bundled sys/lib copy wins over the system one exactly as it does
# at run time.
set -eu

cd "$(dirname "$0")/../ui"
webkit=$(node -e "process.stdout.write(require('path').dirname(require('@playwright/test').webkit.executablePath()))")
checked=0
for dir in "$webkit"/minibrowser-*; do
  for bin in "$dir"/bin/*NetworkProcess; do
    [ -x "$bin" ] || continue
    lib=$(LD_LIBRARY_PATH="$dir/lib:$dir/sys/lib" ldd "$bin" | awk '$1 == "libsoup-3.0.so.0" { print $3 }')
    if [ -z "$lib" ] || [ ! -f "$lib" ]; then
      echo "::error::$bin does not resolve libsoup-3.0.so.0"
      exit 1
    fi
    version=$(grep -a -o -m1 'libsoup/3\.[0-9.]*' "$lib" | head -1)
    echo "$bin -> $lib ($version)"
    case $version in
      libsoup/3.6.5)
        echo "::error::WebKit loads libsoup 3.6.5 (use-after-free, microsoft/playwright#42803)"
        exit 1
        ;;
      libsoup/3.*) ;;
      *)
        echo "::error::cannot read the libsoup version from $lib"
        exit 1
        ;;
    esac
    checked=$((checked + 1))
  done
done

if [ "$checked" -eq 0 ]; then
  echo "::error::no WebKit network process found under $webkit"
  exit 1
fi
