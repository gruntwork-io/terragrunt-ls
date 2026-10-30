#!/usr/bin/env bash

set -euo pipefail

# Check the module for vulnerabilities that its code can reach.
#
# Usage: govulncheck.sh [base-commit]
#
# Without a base commit, fail on every reachable vulnerability. With one, fail
# only on reachable vulnerabilities that the base commit does not have, so a
# pull request fails for what it adds and not for what main already has.

base="${1:-}"

if [[ -z "$base" ]]; then
	exec govulncheck ./...
fi

# Print the IDs of the reachable vulnerabilities in the module at $1. A finding
# is reachable when its trace starts in a function.
reachable() {
	(cd "$1" && govulncheck -format json ./...) |
		jq -r 'select(.finding.trace[0].function != null) | .finding.osv' |
		sort -u
}

base_dir=$(mktemp -d)
trap 'git worktree remove --force "$base_dir"' EXIT

git worktree add --detach "$base_dir" "$base" >/dev/null

# Assign each scan first: a failure inside <(...) would not stop the script.
base_ids=$(reachable "$base_dir")
head_ids=$(reachable .)
added=$(comm -13 <(printf '%s\n' "$base_ids") <(printf '%s\n' "$head_ids"))

if [[ -z "$added" ]]; then
	echo "No reachable vulnerabilities added since $base."
	exit 0
fi

echo "Reachable vulnerabilities added since $base:" >&2
for id in $added; do
	echo "  https://pkg.go.dev/vuln/$id" >&2
done
echo "Run govulncheck ./... for the call paths." >&2
exit 1
