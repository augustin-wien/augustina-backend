#!/bin/sh
# Runs all tests with coverage over all packages and fails when the total is below
# the minimum. Generated code (ent, docs) and main are not measured.
# Usage (from app/): scripts/coverage.sh [minimum percent] [profile]
set -e
export LC_ALL=C
min=${1:-89.5}
profile=${2:-cover.out}
module=$(go list -m)
pkgs=$(go list ./... | grep -v -e "^$module/ent" -e "^$module/docs$" -e "^$module\$" | paste -sd, -)

go test ./... -p 1 -cover -covermode=atomic -coverpkg="$pkgs" -coverprofile="$profile"

# Each test binary writes its own counts for the same blocks; a block counts as
# covered if any binary covered it
total=$(awk 'NR > 1 { key = $1; if (!(key in hit) || $3 > hit[key]) hit[key] = $3; stmts[key] = $2 }
	END { for (k in hit) { all += stmts[k]; if (hit[k] > 0) cov += stmts[k] } printf "%.2f", 100 * cov / all }' "$profile")
echo "Total coverage: $total% (minimum $min%)"
awk -v t="$total" -v m="$min" 'BEGIN { exit (t + 0 < m + 0) }' || { echo "Coverage is below the minimum"; exit 1; }
