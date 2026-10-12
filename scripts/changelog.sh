#!/usr/bin/env bash
# Prepends a section to CHANGELOG.local.md describing the git changes since the
# last `make push`. The last pushed commits are remembered in .last-push.
# Both files are gitignored.
set -euo pipefail

cd "$(dirname "$0")/.."

CHANGELOG=CHANGELOG.local.md
STATE=.last-push
FRONTEND=docker/augustin-frontend
FIRST_RUN_LIMIT=30

last_backend=""
last_frontend=""
if [ -f "$STATE" ]; then
	# shellcheck source=/dev/null
	. "./$STATE"
fi

# section <repo-dir> <title> <last-pushed-commit>
section() {
	local dir=$1 title=$2 last=$3 range log
	echo "### $title"
	echo
	if [ -n "$last" ] && git -C "$dir" cat-file -e "$last^{commit}" 2>/dev/null; then
		range="$last..HEAD"
		log=$(git -C "$dir" log --no-merges --format='- %h %s (%an, %ad)' --date=short "$range")
	else
		echo "_No previous push recorded, showing the last $FIRST_RUN_LIMIT commits._"
		echo
		log=$(git -C "$dir" log --no-merges --format='- %h %s (%an, %ad)' --date=short -n "$FIRST_RUN_LIMIT")
	fi
	if [ -n "$log" ]; then
		echo "$log"
	else
		echo "_No new commits._"
	fi
	local dirty
	dirty=$(git -C "$dir" status --short --ignore-submodules=all)
	if [ -n "$dirty" ]; then
		echo
		echo "Uncommitted changes included in this push:"
		echo
		echo '```'
		echo "$dirty"
		echo '```'
	fi
	echo
}

backend_head=$(git rev-parse HEAD)
frontend_head=$(git -C "$FRONTEND" rev-parse HEAD)
version=$(sed -n 's/^var version = "\(.*\)"/\1/p' app/config/config.go)

entry=$(
	echo "## $(date '+%Y-%m-%d %H:%M') — backend $version"
	echo
	section . "Backend ($(git rev-parse --short HEAD))" "$last_backend"
	section "$FRONTEND" "Frontend ($(git -C "$FRONTEND" rev-parse --short HEAD))" "$last_frontend"
)

{
	echo "$entry"
	echo
	[ -f "$CHANGELOG" ] && cat "$CHANGELOG"
} >"$CHANGELOG.tmp"
mv "$CHANGELOG.tmp" "$CHANGELOG"

cat >"$STATE" <<EOF
last_backend=$backend_head
last_frontend=$frontend_head
EOF

echo "Changelog written to $CHANGELOG"
