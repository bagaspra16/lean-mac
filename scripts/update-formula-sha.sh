#!/usr/bin/env bash
# Fetch the tag tarball, compute its sha256, and rewrite Formula/lm.rb in place.
#
# Usage: ./scripts/update-formula-sha.sh v0.1.0
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: $0 <version>  (e.g. v0.1.0)" >&2
  exit 1
fi

REPO="bagaspra16/lean-mac"
URL="https://github.com/${REPO}/archive/refs/tags/${VERSION}.tar.gz"
FORMULA="$(dirname "$0")/../Formula/lm.rb"

echo "fetching $URL"
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
curl -fsSL "$URL" -o "$TMP"

SHA=$(shasum -a 256 "$TMP" | awk '{print $1}')
echo "sha256: $SHA"

# rewrite url + sha in the formula
sed -i.bak \
  -e "s|archive/refs/tags/v[0-9][^\"]*\.tar\.gz|archive/refs/tags/${VERSION}.tar.gz|" \
  -e "s|REPLACE_WITH_SHA256_AFTER_TAGGING|${SHA}|" \
  -e "s|sha256 \"[0-9a-f]\{64\}\"|sha256 \"${SHA}\"|" \
  "$FORMULA"
rm -f "${FORMULA}.bak"

echo "updated $FORMULA"
echo
echo "next:"
echo "  git add Formula/lm.rb && git commit -m 'formula: ${VERSION}'"
echo "  # push Formula/lm.rb to your tap repo: github.com/bagaspra16/homebrew-lean-mac"
