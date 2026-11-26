#!/bin/sh
# Runs `wikai check` on the three reference bases with configs kept outside the repository.
#
# Usage: sh acceptance/run.sh
#   WIKAI_ACCEPTANCE_CONFIGS  folder holding brain.yaml, am-home.yaml, dev-home.yaml
#   WIKAI_BRAIN, WIKAI_AM_HOME, WIKAI_DEV_HOME  base folders
set -eu

CONFIGS="${WIKAI_ACCEPTANCE_CONFIGS:-$HOME/.config/wikai-acceptance}"
BRAIN="${WIKAI_BRAIN:-$HOME/Brain}"
AM_HOME="${WIKAI_AM_HOME:-$HOME/Development/Pillow/am-home}"
DEV_HOME="${WIKAI_DEV_HOME:-$HOME/Development/Digitaleo/dev-home}"

cd "$(dirname "$0")/.."

for name in brain am-home dev-home; do
    if [ ! -f "$CONFIGS/$name.yaml" ]; then
        printf 'run.sh: missing config %s\n' "$CONFIGS/$name.yaml" >&2
        exit 2
    fi
done

go build -o wikai ./cmd/wikai

status=0
run() {
    printf '== %s (%s)\n' "$1" "$2"
    ./wikai check --config "$CONFIGS/$1.yaml" --dir "$2" || status=1
}

run brain "$BRAIN"
run am-home "$AM_HOME"
run dev-home "$DEV_HOME"

exit "$status"
