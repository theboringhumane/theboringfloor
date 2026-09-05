#!/bin/sh
# theboringfloor — install the TheBoringMajdoor commit-msg attribution hook
# into any git repo.
#
# Normally unnecessary: the office auto-installs this hook into the repo it
# boots in whenever attribution is on (the default — opt out with
# "attribution": "off" in ~/.theboringfloor/configs/brain.json). This script
# is for repos the office never boots in.
#
#   scripts/install-majdoor-hook.sh [path-to-repo]          (default: .)
#   scripts/install-majdoor-hook.sh --uninstall [path-to-repo]
#
# Curl-pipe works too (the hook body is fetched when there's no checkout
# next to this script):
#   curl -fsSL \
#     https://raw.githubusercontent.com/theboringhumane/theboringfloor/main/scripts/install-majdoor-hook.sh \
#     | sh -s -- /path/to/repo
#
# The hook stamps every commit with exactly one
#   Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>
# trailer. Docs: README.md → "Commit attribution — TheBoringMajdoor".
set -eu

RAW_BASE="https://raw.githubusercontent.com/theboringhumane/theboringfloor/main"
MARKER='themajdoor@theboring\.name'

UNINSTALL=0
TARGET=""
TMPD=""

info() { printf '%s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

cleanup() { if [ -n "$TMPD" ] && [ -d "$TMPD" ]; then rm -rf "$TMPD"; fi; }
trap cleanup EXIT

usage() {
    cat <<'USAGE'
install-majdoor-hook — stamp commits with the TheBoringMajdoor co-author trailer

Normally unnecessary: the office auto-installs this hook into the repo it
boots in when attribution is on (the default). This script is for repos the
office never boots in.

Usage:
  scripts/install-majdoor-hook.sh [path-to-repo]          (default: current dir)
  scripts/install-majdoor-hook.sh --uninstall [path-to-repo]

What it does:
  Installs commit-msg into the repo's REAL hooks dir (resolved via
  `git rev-parse --git-path hooks`, so core.hooksPath and worktrees work).
  The hook appends exactly one
      Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>
  trailer to every commit message. A pre-existing commit-msg hook that is
  not ours is backed up to commit-msg.bak-majdoor first; --uninstall removes
  our hook and restores that backup.
USAGE
}

while [ $# -gt 0 ]; do
    case "$1" in
        --uninstall) UNINSTALL=1 ;;
        -h|--help)   usage; exit 0 ;;
        -*)          usage >&2; die "unknown flag: $1" ;;
        *)           [ -z "$TARGET" ] || die "only one repo path accepted"
                     TARGET=$1 ;;
    esac
    shift
done
[ -n "$TARGET" ] || TARGET="."

command -v git >/dev/null 2>&1 || die "git is required on PATH"

# Repo top-level (dies outside a work tree) and the REAL hooks dir — never
# assume .git/hooks: core.hooksPath and linked worktrees move it.
TOP=$(cd "$TARGET" 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null) \
    || die "not a git repository: ${TARGET}"
# (if/else, not case: bash 3.2 /bin/sh misparses `case` inside $(...).)
HOOKS_DIR=$(
    cd "$TOP" || exit 1
    p=$(git rev-parse --git-path hooks) || exit 1
    if [ "${p#/}" != "$p" ]; then
        printf '%s\n' "$p"
    else
        printf '%s/%s\n' "$(pwd -P)" "$p"
    fi
) || die "could not resolve the hooks directory for ${TARGET}"
HOOK_PATH="${HOOKS_DIR}/commit-msg"
BACKUP_PATH="${HOOKS_DIR}/commit-msg.bak-majdoor"

# The hook body normally rides next to this script (repo checkout). When we
# were curl-piped there is no sibling — fetch it from the raw URL instead.
SELF_DIR=$(CDPATH= cd "$(dirname "$0")" 2>/dev/null && pwd -P || printf '')
HOOK_SRC=""
if [ -n "$SELF_DIR" ] && [ -f "${SELF_DIR}/majdoor-commit-msg-hook.sh" ]; then
    HOOK_SRC="${SELF_DIR}/majdoor-commit-msg-hook.sh"
elif [ "$UNINSTALL" -eq 0 ]; then
    TMPD=$(mktemp -d "${TMPDIR:-/tmp}/majdoor-hook.XXXXXX") \
        || die "could not create a temp directory"
    url="${RAW_BASE}/scripts/majdoor-commit-msg-hook.sh"
    HOOK_SRC="${TMPD}/majdoor-commit-msg-hook.sh"
    info "    no checkout copy of the hook next to this script — fetching ${url}"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 -o "$HOOK_SRC" "$url" || die "could not fetch ${url}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$HOOK_SRC" "$url" || die "could not fetch ${url}"
    else
        die "need the hook body next to this script, or curl/wget on PATH to fetch it"
    fi
fi

# Ours = the installed hook carries our email marker (case-insensitive).
is_ours() { [ -f "$HOOK_PATH" ] && grep -qiE "$MARKER" "$HOOK_PATH"; }

do_install() {
    [ -n "$HOOK_SRC" ] || die "internal error: no hook source resolved"
    mkdir -p "$HOOKS_DIR"
    if [ -f "$HOOK_PATH" ]; then
        if is_ours; then
            info "    existing commit-msg hook is already ours — refreshing"
        elif [ -f "$BACKUP_PATH" ]; then
            info "    pre-existing commit-msg hook is NOT ours"
            info "    earlier backup kept untouched at: ${BACKUP_PATH}"
        else
            cp "$HOOK_PATH" "$BACKUP_PATH"
            info "    pre-existing commit-msg hook is NOT ours — backed up to: ${BACKUP_PATH}"
        fi
    fi
    cp "$HOOK_SRC" "$HOOK_PATH"
    chmod 755 "$HOOK_PATH"
    info "    installed: ${HOOK_PATH} (chmod 755)"
}

do_uninstall() {
    if [ ! -f "$HOOK_PATH" ]; then
        if [ -f "$BACKUP_PATH" ]; then
            mv -f "$BACKUP_PATH" "$HOOK_PATH"
            info "    no commit-msg hook present; restored backup to: ${HOOK_PATH}"
        else
            info "    no commit-msg hook at ${HOOK_PATH} — nothing to do"
        fi
        return 0
    fi
    is_ours || die "commit-msg hook at ${HOOK_PATH} is NOT ours — refusing to remove it"
    rm -f "$HOOK_PATH"
    info "    removed: ${HOOK_PATH}"
    if [ -f "$BACKUP_PATH" ]; then
        mv -f "$BACKUP_PATH" "$HOOK_PATH"
        info "    restored pre-majdoor hook from backup: ${HOOK_PATH}"
    fi
}

info ""
info "==> TheBoringMajdoor commit attribution hook"
info "    repo:       ${TOP}"
info "    hooks dir:  ${HOOKS_DIR}"
info "    trailer:    Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>"
if [ "$UNINSTALL" -eq 1 ]; then
    do_uninstall
    info "    done — commits here are no longer stamped."
else
    do_install
    info "    done — every commit here now carries the majdoor's trailer."
fi
