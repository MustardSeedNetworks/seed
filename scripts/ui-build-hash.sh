#!/bin/sh
set -eu

ui_dir=${1:-internal/api/ui}

if [ ! -d "$ui_dir" ] || ! find "$ui_dir" -type f ! -name .gitkeep -print -quit | grep -q .; then
  printf '%s\n' unknown
  exit 0
fi

if command -v md5sum >/dev/null 2>&1; then
  md5_file() { md5sum "$1" | cut -d ' ' -f 1; }
  md5_stdin() { md5sum | cut -d ' ' -f 1; }
elif command -v md5 >/dev/null 2>&1; then
  md5_file() { md5 -q "$1"; }
  md5_stdin() { md5 -q; }
else
  printf '%s\n' 'no MD5 implementation found' >&2
  exit 1
fi

find "$ui_dir" -type f ! -name .gitkeep -print \
  | LC_ALL=C sort \
  | while IFS= read -r file; do
      printf '%s  %s\n' "$(md5_file "$file")" "$file"
    done \
  | md5_stdin
