#!/usr/bin/env bash

# Compiles the Zed extension's tree-sitter queries against the HCL grammar commit pinned in
# zed-extension/extension.toml. tree-sitter rejects a query that names a node the grammar lacks,
# which Zed would otherwise drop at runtime without an error.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZED_DIR="$SCRIPT_DIR/../../zed-extension"
MANIFEST="$ZED_DIR/extension.toml"

repository=$(sed -n '/^\[grammars\.hcl\]/,/^\[/s/^repository = "\(.*\)"$/\1/p' "$MANIFEST")
commit=$(sed -n '/^\[grammars\.hcl\]/,/^\[/s/^commit = "\(.*\)"$/\1/p' "$MANIFEST")
if [ -z "$repository" ] || [ -z "$commit" ]; then
	echo "$MANIFEST has no [grammars.hcl] repository and commit" >&2
	exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

git init -q "$work/grammar"
git -C "$work/grammar" fetch -q --depth 1 "$repository" "$commit"
git -C "$work/grammar" checkout -q FETCH_HEAD
tree-sitter build --output "$work/hcl.so" "$work/grammar"

cat >"$work/terragrunt.hcl" <<'EOF'
locals {
  name = "example"
}

inputs = {
  name = local.name
}
EOF

for query in "$ZED_DIR"/languages/terragrunt/*.scm; do
	echo "Checking ${query#"$ZED_DIR/"}"
	tree-sitter query --quiet --lib-path "$work/hcl.so" --lang-name hcl "$query" "$work/terragrunt.hcl"
done
