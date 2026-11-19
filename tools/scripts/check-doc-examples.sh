#!/usr/bin/env bash
# Checks the Go examples in README.md and docs/:
#   1. no v1 import paths ("github.com/reubenmiller/go-c8y/pkg/...")
#   2. every ```go block that starts with "package main" builds against this module
set -euo pipefail
cd "$(dirname "$0")/../.."

files=(README.md docs/*.md)
status=0

if grep -n '"github.com/reubenmiller/go-c8y/pkg' "${files[@]}"; then
	echo "error: v1 import paths found above; use github.com/reubenmiller/go-c8y/v2/pkg/..." >&2
	status=1
fi

# Blocks are built inside the module so they resolve the local packages.
work=$(mktemp -d "$PWD/internal/docexamples.XXXXXX")
trap 'rm -rf "$work"' EXIT

count=0
for f in "${files[@]}"; do
	fi=$((${fi:-0} + 1))
	awk -v out="$work" -v src="$f" -v fi="$fi" '
		/^[[:space:]]*```(go|golang)[[:space:]]*$/ { match($0, /^[[:space:]]*/); indent = RLENGTH; inblock = 1; body = ""; first = ""; start = NR; next }
		inblock && /^[[:space:]]*```[[:space:]]*$/ {
			inblock = 0
			if (first ~ /^package main/) {
				n++; dir = out "/" fi "_" n; system("mkdir -p " dir)
				file = dir "/main.go"
				printf "%s", body > file; close(file)
				print src ":" start > (dir "/SOURCE"); close(dir "/SOURCE")
			}
			next
		}
		inblock {
			line = substr($0, 1, indent) ~ /^[[:space:]]*$/ ? substr($0, indent + 1) : $0
			if (first == "" && line !~ /^[[:space:]]*$/) first = line
			body = body line "\n"
		}
	' "$f"
done

for dir in "$work"/*/; do
	[ -f "$dir/main.go" ] || continue
	count=$((count + 1))
	if ! out=$(go build -o /dev/null "./${dir#"$PWD"/}" 2>&1); then
		echo "error: example at $(cat "$dir/SOURCE") does not build:" >&2
		echo "$out" | sed 's/^/    /' >&2
		status=1
	fi
done

echo "checked $count package main example(s)"
exit $status
