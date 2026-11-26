#!/bin/sh
# Base paths come only from the environment, so that no base detail lives in the repository.
set -eu

require() {
    if [ -z "$2" ]; then
        printf 'run.sh: %s is not set\n' "$1" >&2
        exit 2
    fi
}
require WIKAI_BRAIN "${WIKAI_BRAIN:-}"
require WIKAI_AM_HOME "${WIKAI_AM_HOME:-}"
require WIKAI_DEV_HOME "${WIKAI_DEV_HOME:-}"

CONFIGS="${WIKAI_ACCEPTANCE_CONFIGS:-$HOME/.config/wikai-acceptance}"

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

run brain "$WIKAI_BRAIN"
run am-home "$WIKAI_AM_HOME"
run dev-home "$WIKAI_DEV_HOME"

exit "$status"
