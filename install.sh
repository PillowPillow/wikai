#!/bin/sh
# Installs wikai from the GitHub release archives, verifying the sha256 checksum
# and refusing to run without a checksum tool.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/PillowPillow/wikai/main/install.sh | sh
#   curl -fsSL ... | WIKAI_VERSION=v1.0.1 sh      # pin a version
#   curl -fsSL ... | WIKAI_INSTALL_DIR=~/bin sh   # override the destination
# (the assignment binds to sh, not curl: a prefix on curl never reaches the script)
set -eu

REPO="PillowPillow/wikai"
INSTALL_DIR="${WIKAI_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
    printf 'install.sh: %s\n' "$1" >&2
    exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required — install it with your package manager and re-run"
command -v tar >/dev/null 2>&1 || fail "tar is required — install it with your package manager and re-run"
command -v mktemp >/dev/null 2>&1 || fail "mktemp is required — install it with your package manager and re-run"
command -v install >/dev/null 2>&1 || fail "install is required — install it with your package manager and re-run"

case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) fail "unsupported OS '$(uname -s)' — releases cover darwin and linux (amd64/arm64) only; see https://github.com/$REPO/releases" ;;
esac

case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) fail "unsupported architecture '$(uname -m)' — releases cover amd64 and arm64 only; see https://github.com/$REPO/releases" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
    checksum_tool=sha256sum
elif command -v shasum >/dev/null 2>&1; then
    checksum_tool=shasum
else
    fail "neither sha256sum nor shasum is available — refusing to install an unverified binary; install coreutils and re-run"
fi

tag="${WIKAI_VERSION:-}"
if [ -z "$tag" ]; then
    # the /releases/latest redirect names the tag, which avoids api.github.com rate limits and a jq dependency
    final_url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") \
        || fail "cannot resolve the latest release from github.com — check your network, or pin one with WIKAI_VERSION=v1.0.1"
    tag="${final_url##*/}"
    case "$tag" in
        v*) ;;
        *) fail "could not parse a release tag from '$final_url' — pin one with WIKAI_VERSION=v1.0.1" ;;
    esac
else
    case "$tag" in
        v*) ;;
        *) fail "WIKAI_VERSION='$tag' must name a tag like v1.0.1 — the tags carry the leading v; see https://github.com/$REPO/releases" ;;
    esac
fi

# the archive name uses the version without the leading v, as .goreleaser.yaml's name_template does
archive="wikai_${tag#v}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d)
# the staging file is removed with `rm -f` on its own: $INSTALL_DIR is caller-supplied, so cleanup never recurses into it
trap 'rm -rf "$tmp"; rm -f "$INSTALL_DIR/.wikai.new.$$"' EXIT INT TERM

printf 'Downloading wikai %s (%s/%s)...\n' "$tag" "$os" "$arch"
curl -fsSL -o "$tmp/$archive" "$base_url/$archive" \
    || fail "download failed for $base_url/$archive — does release $tag exist and ship $os/$arch?"
curl -fsSL -o "$tmp/checksums.txt" "$base_url/checksums.txt" \
    || fail "download failed for $base_url/checksums.txt"

# checksums.txt comes from the same release: it proves integrity, not authenticity
grep " $archive\$" "$tmp/checksums.txt" > "$tmp/expected.txt" \
    || fail "checksums.txt of $tag has no entry for $archive — the release layout changed; report this at https://github.com/$REPO/issues"
case "$checksum_tool" in
    sha256sum) (cd "$tmp" && sha256sum -c expected.txt >/dev/null 2>&1) ;;
    shasum) (cd "$tmp" && shasum -a 256 -c expected.txt >/dev/null 2>&1) ;;
esac || fail "checksum mismatch for $archive — the download is corrupted or incomplete; re-run, and report it if this persists"

tar -xzf "$tmp/$archive" -C "$tmp" wikai \
    || fail "could not extract 'wikai' from $archive — the release layout changed; report this at https://github.com/$REPO/issues"

mkdir -p "$INSTALL_DIR" || fail "cannot create $INSTALL_DIR — pick another destination with WIKAI_INSTALL_DIR"
# install beside the target then rename: the swap is atomic and works while wikai is running
# shellcheck disable=SC2015
install -m 755 "$tmp/wikai" "$INSTALL_DIR/.wikai.new.$$" \
    && mv -f "$INSTALL_DIR/.wikai.new.$$" "$INSTALL_DIR/wikai" \
    || fail "cannot install to $INSTALL_DIR/wikai — check the directory's permissions, or pick another destination with WIKAI_INSTALL_DIR"

printf 'Installed %s/wikai\n' "$INSTALL_DIR"
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        printf 'warning: %s is not in your PATH — add this line to your shell rc (~/.bashrc, ~/.zshrc, ...):\n' "$INSTALL_DIR" >&2
        # the line is pasted into a shell rc, where $PATH must stay unexpanded
        # shellcheck disable=SC2016
        printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR" >&2
        ;;
esac

"$INSTALL_DIR/wikai" --version \
    || fail "installed, but $INSTALL_DIR/wikai does not run on this machine — a noexec mount or a binary for another architecture; pick another destination with WIKAI_INSTALL_DIR, or see https://github.com/$REPO/releases"
