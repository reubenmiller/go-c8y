#!/usr/bin/env bash
# Fails when a public package exposes a resty type in an exported signature or
# field, or embeds the internal HTTP core (which would promote its client). Comment lines (indented by 4 spaces in `go doc` output) are ignored.
# The only allowed exception is Client.Resty(), the documented escape hatch.
set -euo pipefail

status=0
for pkg in $(go list ./pkg/... | grep -v '/internal/'); do
	matches=$(go doc -all "$pkg" 2>/dev/null \
		| grep -E 'resty\.' \
		| grep -vE '^    ' \
		| grep -vE '^func \(c \*Client\) Resty\(\) \*resty\.Client$' || true)
	if [ -n "$matches" ]; then
		echo "resty type in public API of $pkg:"
		echo "$matches" | sed 's/^/  /'
		status=1
	fi
	embedded=$(go doc -all "$pkg" 2>/dev/null \
		| grep -E $'(struct ?\\{ ?httpcore\\.|^\t+httpcore\\.[A-Za-z]+[[:space:]]*$)' || true)
	if [ -n "$embedded" ]; then
		echo "internal httpcore type embedded in public API of $pkg:"
		echo "$embedded" | sed 's/^/  /'
		status=1
	fi
done

if [ "$status" -eq 0 ]; then
	echo "No resty types or embedded httpcore types in the public API."
fi
exit "$status"
