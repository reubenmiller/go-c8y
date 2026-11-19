#!/usr/bin/env bash
# Compares the exported API with the latest v2 release tag using gorelease.
# Before v2.0.0 exists the result is informational only (exit 0); once a
# v2.x.y release tag exists, an incompatible change fails the check.
set -euo pipefail
cd "$(dirname "$0")/../.."

git fetch --quiet --tags origin 2>/dev/null || true
base=$(git tag -l 'v2.*' --sort=-v:refname | grep -E '^v2\.[0-9]+\.[0-9]+$' | head -1 || true)
blocking=true
if [ -z "$base" ]; then
	base=$(git tag -l 'v2.*' --sort=-v:refname | head -1 || true)
	blocking=false
fi
if [ -z "$base" ]; then
	echo "No v2 tag yet; skipping the API compatibility check."
	exit 0
fi

echo "Comparing with $base (blocking: $blocking)"
if go run golang.org/x/exp/cmd/gorelease@latest -base="$base"; then
	exit 0
fi
if [ "$blocking" = true ]; then
	echo "error: incompatible API change compared with $base" >&2
	exit 1
fi
echo "warning: incompatible changes compared with pre-release $base (not blocking before v2.0.0)"
