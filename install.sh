#!/bin/sh
# theboringoffice installer — curl-pipe friendly.
#
#   curl -fsSL https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh
#
# Flags (no interactive prompts — everything is flag-driven):
#   --dry-run            Print every action without executing it
#   --prefix DIR         Binary install prefix (default: /usr/local/bin if writable,
#                        else ~/.local/bin with a PATH hint)
#   --backend NAME       LLM transport the office boots on: opencode (default)
#                        or claudecode (needs the claude CLI; warn-only when absent)
#   --majdoor-hook DIR   Also install the TheBoringMajdoor commit-msg attribution
#                        hook into the repo at DIR (opt-in)
#   --skip-agentmemory   Do not install/start the agentmemory background service
#   --uninstall          Remove the theboringoffice binary and the agentmemory service
#
# Consumes goreleaser assets from https://github.com/theboringhumane/theboringoffice/releases :
#   theboringoffice_<version>_<os>_<arch>.tar.gz        (contains a single binary: theboringoffice)
#   theboringoffice_<version>_checksums.txt
# Fallback when the GitHub API can't resolve a version:
#   .../releases/latest/download/theboringoffice_<os>_<arch>.tar.gz   (checksums skipped, loudly)

set -eu
umask 022

APP="theboringoffice"
REPO="theboringhumane/theboringoffice"
REPO_URL="https://github.com/${REPO}"
API_LATEST="https://api.github.com/repos/${REPO}/releases/latest"
RAW_BASE="https://raw.githubusercontent.com/${REPO}/main"

PLIST_LABEL="ai.agentmemory.server"
PLIST_PATH="${HOME}/Library/LaunchAgents/${PLIST_LABEL}.plist"
SYSTEMD_UNIT_DIR="${HOME}/.config/systemd/user"
SYSTEMD_UNIT_PATH="${SYSTEMD_UNIT_DIR}/agentmemory.service"
AM_LOG_DIR="${HOME}/.agentmemory/logs"

DRY_RUN=0
SKIP_AGENTMEMORY=0
UNINSTALL=0
PREFIX=""
PATH_HINT=0
BACKEND="opencode"
BACKEND_STATE=""
MAJDOOR_HOOK_DIR=""
MAJDOOR_STATE="not requested"
STAGE_NUM=0
OS=""
ARCH=""
VERSION=""
TARBALL=""
CHECKSUMS=""
DL_BASE=""
TMPWORK=""
AM_BIN=""
AM_SERVICE_STATE="not attempted"

# ---------------------------------------------------------------- utilities

info() { printf '%s\n' "$*"; }
warn() { printf '  ! %s\n' "$*" >&2; }
die() {
    printf '\nERROR: %s\n' "$*" >&2
    exit 1
}

# run — execute a mutating command, or just narrate it in dry-run mode.
run() {
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '  [dry-run] %s\n' "$*"
        return 0
    fi
    "$@"
}

jokes_pool() {
    cat <<'JOKES'
Why do programmers prefer dark mode? Because light attracts bugs.
There are 10 kinds of people: those who understand binary and those who don't.
A SQL query walks into a bar and sees two tables... and asks, "Mind if I JOIN you?"
Why did the developer go broke? He used up all his cache.
Debugging: being the detective in a crime film where you are also the murderer.
Why do Java developers wear glasses? Because they don't C#.
A QA engineer walks into a bar. Orders 1 beer. Orders 0 beers. Orders 99999999 beers. Orders a lizard. Orders -1 beers.
To understand what recursion is, you must first understand recursion.
JOKES
}

stage() {
    STAGE_NUM=$((STAGE_NUM + 1))
    idx=$((($(date +%s) + STAGE_NUM) % 8 + 1))
    joke=$(jokes_pool | sed -n "${idx}p")
    printf '\n'
    printf '==> [%d] %s\n' "$STAGE_NUM" "$1"
    printf '    joke %d/8: %s\n' "$idx" "$joke"
}

print_banner() {
# "theboringoffice" is 14 chars — too wide for one figlet row at 80 cols,
# so the wordmark splits into two stacked words. Generator discipline
# (same as the original grafeio banner): figlet standard, full-width
# layout — `figlet -f standard -h full theboring` / `... office`, every
# row <= 80 cols (widest: 64).
    cat <<'BANNER'

  _     _              _                      _
 | |_  | |__     ___  | |__     ___    _ __  (_)  _ __     __ _
 | __| | '_ \   / _ \ | '_ \   / _ \  | '__| | | | '_ \   / _` |
 | |_  | | | | |  __/ | |_) | | (_) | | |    | | | | | | | (_| |
  \__| |_| |_|  \___| |_.__/   \___/  |_|    |_| |_| |_|  \__, |
                                                          |___/
           __    __   _
   ___    / _|  / _| (_)   ___    ___
  / _ \  | |_  | |_  | |  / __|  / _ \
 | (_) | |  _| |  _| | | | (__  |  __/
  \___/  |_|   |_|   |_|  \___|  \___|

        t h e b o r i n g o f f i c e  -  i n s t a l l e r
        github.com/theboringhumane/theboringoffice
BANNER
    printf '\n'
}

usage() {
    cat <<'USAGE'
theboringoffice installer

Usage:
  sh install.sh [--dry-run] [--prefix DIR] [--skip-agentmemory] [--uninstall]
  curl -fsSL https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh

Flags:
  --dry-run            Print every action without executing anything
  --prefix DIR         Install prefix for the theboringoffice binary
                       (default: /usr/local/bin if writable, else ~/.local/bin)
  --skip-agentmemory   Do not install/start the agentmemory background service
  --backend NAME       LLM transport: opencode (default) | claudecode
  --majdoor-hook DIR   Install the TheBoringMajdoor commit-msg attribution
                       hook into the repo at DIR (opt-in)
  --uninstall          Remove the theboringoffice binary and the agentmemory service
USAGE
}

cleanup() {
    if [ -n "$TMPWORK" ] && [ -d "$TMPWORK" ]; then
        case "$TMPWORK" in
            */theboringoffice-install.*) rm -rf "$TMPWORK" ;;
        esac
    fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# ---------------------------------------------------------------- detection

detect_platform() {
    stage "Detect platform"
    s=$(uname -s)
    case "$s" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux" ;;
        *)      unsupported_os "$s" ;;
    esac
    m=$(uname -m)
    case "$m" in
        x86_64)          ARCH="amd64" ;;
        amd64)           ARCH="amd64" ;;
        arm64|aarch64)   ARCH="arm64" ;;
        *)               unsupported_arch "$m" ;;
    esac
    info "    OS:   ${OS}"
    info "    arch: ${ARCH}"
}

unsupported_os() {
    cat <<EOF

  Sorry, '${APP}' release binaries currently ship for macOS and Linux only —
  your OS reports as '$1'.

  Manual options:
    1. Build from source (requires Go):
         git clone ${REPO_URL}
         cd theboringoffice && go build -o theboringoffice ./cmd/theboringoffice
    2. Watch ${REPO_URL}/releases for new platform builds.

EOF
    exit 1
}

unsupported_arch() {
    cat <<EOF

  Sorry, '${APP}' release binaries currently ship for amd64 and arm64 only —
  your machine reports as '$1'.

  Manual options:
    1. Build from source (requires Go):
         git clone ${REPO_URL}
         cd theboringoffice && go build -o theboringoffice ./cmd/theboringoffice
    2. Watch ${REPO_URL}/releases for new architecture builds.

EOF
    exit 1
}

expand_prefix() {
    case "$PREFIX" in
        "~")    PREFIX="$HOME" ;;
        "~/"*)  PREFIX="${HOME}/${PREFIX#"~/"}" ;;
        *)      : ;;
    esac
}

resolve_prefix() {
    if [ -n "$PREFIX" ]; then
        expand_prefix
        info "    prefix: ${PREFIX} (from --prefix)"
    elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
        PREFIX="/usr/local/bin"
        info "    prefix: ${PREFIX} (writable system prefix)"
    else
        PREFIX="${HOME}/.local/bin"
        info "    prefix: ${PREFIX} (/usr/local/bin not writable)"
    fi
    case ":${PATH}:" in
        *":${PREFIX}:"*) PATH_HINT=0 ;;
        *)               PATH_HINT=1 ;;
    esac
}

# ---------------------------------------------------------------- download

fetch() { # $1 = url, $2 = dest file
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '  [dry-run] download %s\n            -> %s\n' "$1" "$2"
        return 0
    fi
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$2" "$1"
    else
        die "need 'curl' or 'wget' on PATH to download theboringoffice"
    fi
}

http_get() { # $1 = url -> stdout, empty on failure
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --max-time 20 "$1" 2>/dev/null || true
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O - "$1" 2>/dev/null || true
    fi
}

resolve_version() {
    stage "Resolve latest release"
    resp=$(http_get "$API_LATEST" || true)
    tag=""
    if [ -n "$resp" ]; then
        tag=$(printf '%s\n' "$resp" | grep '"tag_name"' | head -n 1 \
            | sed -E 's/^[^:]*:[[:space:]]*"v?([^"]+)".*$/\1/') || tag=""
        case "$tag" in
            ""|*tag_name*) tag="" ;;
            *)             : ;;
        esac
    fi
    if [ -n "$tag" ]; then
        VERSION="$tag"
        TARBALL="${APP}_${VERSION}_${OS}_${ARCH}.tar.gz"
        CHECKSUMS="${APP}_${VERSION}_checksums.txt"
        DL_BASE="${REPO_URL}/releases/download/v${VERSION}"
        info "    latest release: v${VERSION}"
        info "    asset:          ${TARBALL}"
    else
        warn "could not resolve a release via the GitHub API (offline, or no releases published yet)"
        warn "falling back to /releases/latest/download/ URL (no version pinning, checksums skipped)"
        VERSION=""
        TARBALL="${APP}_${OS}_${ARCH}.tar.gz"
        CHECKSUMS=""
        DL_BASE="${REPO_URL}/releases/latest/download"
        info "    asset:          ${TARBALL}"
    fi
    if [ "$DRY_RUN" -eq 1 ]; then
        info "    [dry-run] note: real run prefers the API-resolved versioned URL,"
        info "             i.e. ${REPO_URL}/releases/download/v<version>/theboringoffice_<version>_${OS}_${ARCH}.tar.gz"
    fi
}

make_tmpdir() {
    if [ "$DRY_RUN" -eq 1 ]; then
        TMPWORK="(mktemp -d theboringoffice-install.XXXXXX)"
        info "  [dry-run] create temp workdir via mktemp -d (trap-cleanup on exit)"
        return 0
    fi
    TMPWORK=$(mktemp -d "${TMPDIR:-/tmp}/theboringoffice-install.XXXXXX") \
        || die "could not create a temp directory"
    info "    temp workdir: ${TMPWORK}"
}

download_assets() {
    stage "Download theboringoffice"
    make_tmpdir
    fetch "${DL_BASE}/${TARBALL}" "${TMPWORK}/${TARBALL}"
    if [ -n "$CHECKSUMS" ]; then
        fetch "${DL_BASE}/${CHECKSUMS}" "${TMPWORK}/${CHECKSUMS}"
    fi
}

verify_checksum() {
    stage "Verify SHA-256 checksum"
    if [ -z "$CHECKSUMS" ]; then
        warn "no checksums file available on the fallback URL — skipping verification"
        warn "the binary is NOT checksum-verified; re-run once a tagged release exists for full verification"
        return 0
    fi
    tool=""
    if command -v sha256sum >/dev/null 2>&1; then
        tool="sha256sum -c -"
    elif command -v shasum >/dev/null 2>&1; then
        tool="shasum -a 256 -c -"
    fi
    if [ -z "$tool" ]; then
        warn "neither 'sha256sum' nor 'shasum' found on PATH — proceeding WITHOUT checksum verification"
        return 0
    fi
    if [ "$DRY_RUN" -eq 1 ]; then
        info "  [dry-run] (cd workdir && grep '[[:space:]]${TARBALL}\$' ${CHECKSUMS} | ${tool})"
        return 0
    fi
    info "    verifying: ${TARBALL}"
    if command -v sha256sum >/dev/null 2>&1; then
        ( cd "$TMPWORK" && grep "[[:space:]]${TARBALL}\$" "$CHECKSUMS" | sha256sum -c - ) \
            || die "checksum verification FAILED for ${TARBALL}"
    else
        ( cd "$TMPWORK" && grep "[[:space:]]${TARBALL}\$" "$CHECKSUMS" | shasum -a 256 -c - ) \
            || die "checksum verification FAILED for ${TARBALL}"
    fi
    info "    checksum OK"
}

install_binary() {
    stage "Install binary"
    run mkdir -p "$PREFIX"
    if [ "$DRY_RUN" -eq 1 ]; then
        info "  [dry-run] tar -xzf workdir/${TARBALL} -C workdir theboringoffice   (extract ONLY the theboringoffice member)"
        info "  [dry-run] cp workdir/theboringoffice ${PREFIX}/.theboringoffice.tmp.\$PID && chmod 755 <tmp> && mv -f <tmp> ${PREFIX}/theboringoffice   (atomic rename — never an in-place overwrite)"
        return 0
    fi
    tar -xzf "${TMPWORK}/${TARBALL}" -C "$TMPWORK" theboringoffice 2>/dev/null \
        || tar -xzf "${TMPWORK}/${TARBALL}" -C "$TMPWORK"
    [ -f "${TMPWORK}/theboringoffice" ] || die "tarball did not contain a 'theboringoffice' binary"
    if [ -e "${PREFIX}/theboringoffice" ] && [ ! -w "${PREFIX}/theboringoffice" ]; then
        die "${PREFIX}/theboringoffice is not writable — re-run with --prefix ~/.local/bin"
    fi
    # Install via ATOMIC RENAME, never an in-place overwrite: a cp that
    # truncates+rewrites an executable that any process is still running gets
    # that process SIGKILLed by macOS ("Code Signature Invalid", namespace
    # CODESIGNING), and fresh execs of the poisoned vnode die the same way
    # until it is reclaimed. rename(2) swaps the vnode atomically: running
    # instances keep their old inode untouched, new execs always see a
    # complete, intact file — no kill window, no torn binary.
    tmp_dest="${PREFIX}/.theboringoffice.tmp.$$"
    trap 'rm -f "$tmp_dest" 2>/dev/null; cleanup' EXIT
    cp "${TMPWORK}/theboringoffice" "$tmp_dest"
    chmod 755 "$tmp_dest"
    mv -f "$tmp_dest" "${PREFIX}/theboringoffice"
    trap cleanup EXIT
    info "    installed: ${PREFIX}/theboringoffice"
}

# ---------------------------------------------------------------- agentmemory

print_manual_agentmemory() { # $1 = reason
    warn "agentmemory auto-setup unavailable: $1"
    cat <<EOF
    Manual setup (best effort — theboringoffice itself is fully installed):
      1. install  : npm i -g @agentmemory/agentmemory
      2. init env : agentmemory init
      3. run      : agentmemory          (keep alive with tmux, screen, or your init system)
      4. check    : agentmemory status
EOF
}

absolutize() { # $1 = path -> absolute path on stdout
    case "$1" in
        /*) printf '%s\n' "$1" ;;
        *)  ( cd "$(dirname "$1")" && printf '%s/%s\n' "$(pwd -P)" "$(basename "$1")" ) ;;
    esac
}

setup_agentmemory() {
    stage "agentmemory background service"
    if [ "$SKIP_AGENTMEMORY" -eq 1 ]; then
        info "    --skip-agentmemory given; skipping. Re-run without it to configure the service."
        AM_SERVICE_STATE="skipped (--skip-agentmemory)"
        return 0
    fi

    AM_BIN=$(command -v agentmemory 2>/dev/null || true)
    if [ -z "$AM_BIN" ] && command -v npm >/dev/null 2>&1; then
        info "    agentmemory not on PATH — installing via npm"
        # Guarded: errexit is ON, so ANY npm failure (404, network, perms)
        # must degrade to the manual path — never abort the whole install.
        if run npm i -g @agentmemory/agentmemory; then
            if [ "$DRY_RUN" -eq 1 ]; then
                AM_BIN="/path/to/agentmemory (resolved after npm i -g)"
            else
                AM_BIN=$(command -v agentmemory 2>/dev/null || true)
            fi
        else
            print_manual_agentmemory "npm i -g @agentmemory/agentmemory failed (see the npm error above)"
            AM_SERVICE_STATE="not configured (npm install failed — see above)"
            return 0
        fi
    fi
    if [ -z "$AM_BIN" ]; then
        print_manual_agentmemory "agentmemory is not installed and npm is unavailable"
        AM_SERVICE_STATE="not configured (install manually — see above)"
        return 0
    fi
    info "    agentmemory binary: ${AM_BIN}"

    # idempotent: seeds ~/.agentmemory/.env if absent. Guarded so an init
    # failure can never abort the install either (errexit is ON).
    if ! run agentmemory init; then
        warn "agentmemory init failed — continuing; re-run 'agentmemory init' by hand"
    fi

    case "$OS" in
        darwin) setup_launchd ;;
        linux)  setup_systemd ;;
        *)
            print_manual_agentmemory "no launchd or systemd --user on this OS"
            AM_SERVICE_STATE="manual setup required (see above)"
            ;;
    esac
}

setup_launchd() {
    if ! command -v launchctl >/dev/null 2>&1; then
        print_manual_agentmemory "launchctl not found"
        AM_SERVICE_STATE="manual setup required (see above)"
        return 0
    fi
    uid=$(id -u)
    case "$AM_BIN" in
        /*) AM_ABS="$AM_BIN" ;;
        *)  AM_ABS=$(absolutize "$AM_BIN") ;;
    esac
    run mkdir -p "${HOME}/Library/LaunchAgents"
    run mkdir -p "$AM_LOG_DIR"

    if [ "$DRY_RUN" -eq 1 ]; then
        info "  [dry-run] would write ${PLIST_PATH} :"
        {
            cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${PLIST_LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${AM_ABS}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${AM_LOG_DIR}/agentmemory.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>${AM_LOG_DIR}/agentmemory.stderr.log</string>
</dict>
</plist>
EOF
        } | sed 's/^/    | /'
        info "  [dry-run] if already loaded: launchctl bootout gui/${uid} ${PLIST_PATH}"
        info "  [dry-run] launchctl bootstrap gui/${uid} ${PLIST_PATH}"
        info "  [dry-run] (fallback on older macOS: launchctl load -w ${PLIST_PATH})"
        AM_SERVICE_STATE="would install LaunchAgent ${PLIST_LABEL} (dry-run)"
        return 0
    fi

    cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${PLIST_LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${AM_ABS}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${AM_LOG_DIR}/agentmemory.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>${AM_LOG_DIR}/agentmemory.stderr.log</string>
</dict>
</plist>
EOF
    info "    wrote ${PLIST_PATH}"

    # Bootout first if the service is already loaded (reinstall path).
    if launchctl print "gui/${uid}/${PLIST_LABEL}" >/dev/null 2>&1; then
        info "    service already loaded — bootout before reload"
        launchctl bootout "gui/${uid}" "$PLIST_PATH" 2>/dev/null \
            || launchctl unload -w "$PLIST_PATH" 2>/dev/null || true
    fi
    if launchctl bootstrap "gui/${uid}" "$PLIST_PATH" 2>/dev/null; then
        info "    bootstrapped: launchctl bootstrap gui/${uid}"
        AM_SERVICE_STATE="running via launchd (${PLIST_LABEL}, RunAtLoad+KeepAlive)"
    elif launchctl load -w "$PLIST_PATH" 2>/dev/null; then
        info "    loaded via legacy: launchctl load -w"
        AM_SERVICE_STATE="running via launchd (legacy load -w)"
    else
        print_manual_agentmemory "launchctl bootstrap failed"
        AM_SERVICE_STATE="service file written but NOT started — see above"
    fi
}

setup_systemd() {
    if ! command -v systemctl >/dev/null 2>&1 || ! systemctl --user list-unit-files >/dev/null 2>&1; then
        print_manual_agentmemory "systemd --user not available"
        AM_SERVICE_STATE="manual setup required (see above)"
        return 0
    fi
    case "$AM_BIN" in
        /*) AM_ABS="$AM_BIN" ;;
        *)  AM_ABS=$(absolutize "$AM_BIN") ;;
    esac
    run mkdir -p "$SYSTEMD_UNIT_DIR"

    unit_body() {
        cat <<EOF
[Unit]
Description=agentmemory server (theboringoffice companion)

[Service]
ExecStart=${AM_ABS}
Restart=always

[Install]
WantedBy=default.target
EOF
    }

    if [ "$DRY_RUN" -eq 1 ]; then
        info "  [dry-run] would write ${SYSTEMD_UNIT_PATH} :"
        unit_body | sed 's/^/    | /'
        info "  [dry-run] systemctl --user daemon-reload"
        info "  [dry-run] systemctl --user enable --now agentmemory.service"
        info "    hint: sudo loginctl enable-linger ${USER:-$(id -un)}   (survives logout without a login session)"
        AM_SERVICE_STATE="would install systemd user unit agentmemory.service (dry-run)"
        return 0
    fi

    unit_body > "$SYSTEMD_UNIT_PATH"
    info "    wrote ${SYSTEMD_UNIT_PATH}"
    if systemctl --user daemon-reload && systemctl --user enable --now agentmemory.service; then
        AM_SERVICE_STATE="enabled via systemd --user (agentmemory.service, Restart=always)"
        info "    enabled & started: systemctl --user enable --now agentmemory.service"
    else
        print_manual_agentmemory "systemctl --user failed"
        AM_SERVICE_STATE="unit written but NOT started — see above"
    fi
    info "    hint: sudo loginctl enable-linger ${USER:-$(id -un)}   (survives logout without a login session)"
}

# ---------------------------------------------------------------- uninstall

do_uninstall() {
    s=$(uname -s)
    case "$s" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux" ;;
        *)      OS="other" ;;
    esac

    stage "Remove theboringoffice binary"
    if [ -n "$PREFIX" ]; then
        expand_prefix
    elif [ -d /usr/local/bin ] && [ -w "/usr/local/bin" ]; then
        PREFIX="/usr/local/bin"
    else
        PREFIX="${HOME}/.local/bin"
    fi
    if [ -f "${PREFIX}/theboringoffice" ] || [ "$DRY_RUN" -eq 1 ]; then
        run rm -f "${PREFIX}/theboringoffice"
        info "    removed: ${PREFIX}/theboringoffice"
    else
        info "    no binary at ${PREFIX}/theboringoffice — nothing to do"
    fi

    stage "Remove agentmemory service"
    found_service=0
    case "$OS" in
        darwin)
            if [ -f "$PLIST_PATH" ] || [ "$DRY_RUN" -eq 1 ]; then
                found_service=1
                if [ "$DRY_RUN" -eq 1 ]; then
                    info "  [dry-run] launchctl bootout gui/$(id -u) ${PLIST_PATH} (fallback: launchctl unload -w)"
                else
                    launchctl bootout "gui/$(id -u)" "$PLIST_PATH" 2>/dev/null \
                        || launchctl unload -w "$PLIST_PATH" 2>/dev/null || true
                fi
                run rm -f "$PLIST_PATH"
                info "    removed: ${PLIST_PATH}"
            fi
            ;;
        linux)
            if [ -f "$SYSTEMD_UNIT_PATH" ] || [ "$DRY_RUN" -eq 1 ]; then
                found_service=1
                if [ "$DRY_RUN" -eq 1 ]; then
                    info "  [dry-run] systemctl --user disable --now agentmemory.service"
                elif command -v systemctl >/dev/null 2>&1; then
                    systemctl --user disable --now agentmemory.service >/dev/null 2>&1 || true
                fi
                run rm -f "$SYSTEMD_UNIT_PATH"
                if [ "$DRY_RUN" -eq 1 ]; then
                    info "  [dry-run] systemctl --user daemon-reload"
                elif command -v systemctl >/dev/null 2>&1; then
                    systemctl --user daemon-reload >/dev/null 2>&1 || true
                fi
                info "    removed: ${SYSTEMD_UNIT_PATH}"
            fi
            ;;
        *)
            info "    unsupported OS for managed service removal (${s})"
            ;;
    esac
    if [ "$found_service" -eq 0 ] && [ "$DRY_RUN" -eq 0 ]; then
        info "    no ${PLIST_LABEL} / agentmemory.service installation found"
    fi
    info "    note: the agentmemory npm package itself is left installed; remove with: npm rm -g @agentmemory/agentmemory"
}

# ---------------------------------------------------------------- backend

# brain_path — the brain.json the office will boot on: honors the
# THEBORINGOFFICE_HOME / GRAFEIO_HOME scratch-root overrides exactly like the
# binary's own config.Path (install.sh --dry-run proofs point one at a
# scratch home; a real install almost always lands on $HOME).
brain_path() {
    home="${THEBORINGOFFICE_HOME:-${GRAFEIO_HOME:-$HOME}}"
    printf '%s/.theboringoffice/configs/brain.json\n' "$home"
}

# seed_brain_backend — writes backend.name=<BACKEND> into brain.json for the
# default AND the explicit pick alike (an explicit name makes later /backend
# swaps' persisted state legible). An EXISTING backend.name key always wins:
# the installer never silently re-selects a member's transport. Toolchain
# ladder (zero-jq landscape): jq → python3 (append only when the key is
# missing) → plain JSON overwrite from `theboringoffice --print-default-config`
# + sed insert — and every step narrates in dry-run, mutates nothing.
seed_brain_backend() {
    brain="$(brain_path)"
    if [ "$DRY_RUN" -eq 1 ]; then
        info "  [dry-run] seed ${brain} with backend.name=${BACKEND} (existing key preserved; jq > python3 > print-default-config+sed)"
        return 0
    fi
    run mkdir -p "$(dirname "$brain")"
    if [ -f "$brain" ]; then
        if command -v jq >/dev/null 2>&1; then
            cur="$(jq -r '.backend.name // ""' "$brain" 2>/dev/null || true)"
            if [ -n "$cur" ]; then
                info "    brain.json backend.name already set (${cur}) — preserved"
                return 0
            fi
            tmp="${brain}.tmp.$$"
            jq --arg n "$BACKEND" '.backend.name = $n' "$brain" >"$tmp" \
                && mv -f "$tmp" "$brain" \
                && info "    brain.json backend.name → ${BACKEND} (jq)"
            return 0
        fi
        if command -v python3 >/dev/null 2>&1; then
            python3 - "$brain" "$BACKEND" <<'PYEOF'
import json, sys
path, name = sys.argv[1], sys.argv[2]
doc = json.load(open(path))
backend = doc.setdefault("backend", {})
if not backend.get("name"):
    backend["name"] = name
    with open(path, "w") as fh:
        json.dump(doc, fh, indent=2)
        fh.write("\n")
PYEOF
            info "    brain.json backend.name → ${BACKEND} when missing (python3)"
            return 0
        fi
        # No safe in-place editor on this box and a brain.json already
        # exists: it is NEVER overwritten by the fallback (the file wins
        # over an uneditable key — same degrade-open shape as agentmemory).
        warn "no jq/python3 on PATH and brain.json exists — not touching it"
        warn "set \"backend\": { \"name\": \"${BACKEND}\" } by hand, or run /backend ${BACKEND} in-app"
        return 0
    fi
    # No brain.json yet: write the stock default with the name injected.
    cfg="$("${PREFIX}/theboringoffice" --print-default-config 2>/dev/null || true)"
    if [ -z "$cfg" ]; then
        warn "could not read the default brain.json (theboringoffice --print-default-config failed) — first boot writes one; /backend ${BACKEND} switches then"
        return 0
    fi
    if command -v python3 >/dev/null 2>&1; then
        printf '%s\n' "$cfg" | python3 -c 'import json,sys; d=json.load(sys.stdin); d.setdefault("backend",{})["name"]=sys.argv[1]; json.dump(d,sys.stdout,indent=2); sys.stdout.write("\n")' "$BACKEND" >"$brain"
        info "    wrote ${brain} with backend.name=${BACKEND} (python3)"
        return 0
    fi
    if command -v jq >/dev/null 2>&1; then
        printf '%s\n' "$cfg" | jq --arg n "$BACKEND" '.backend.name = $n' >"$brain"
        info "    wrote ${brain} with backend.name=${BACKEND} (jq)"
        return 0
    fi
    # sed insert (zero-jq AND zero-python): inject into the "backend": {
    # block of the stock JSON — the printed default's shape is stable.
    printf '%s\n' "$cfg" | sed 's/^  "backend": {$/  "backend": {\n    "name": "'"$BACKEND"'",/' >"$brain"
    info "    wrote ${brain} with backend.name=${BACKEND} (sed insert)"
}

setup_backend() {
    stage "Backend selection"
    case "$BACKEND" in
        opencode|claudecode) : ;;
        *) die "--backend must be opencode|claudecode" ;;
    esac
    info "    backend: ${BACKEND}   (brain.json backend.name — swap live with /backend in-app)"
    if [ "$BACKEND" = "claudecode" ]; then
        # warn-only detection (never die): the office itself reports the
        # missing CLI on boot and /backend opencode recovers instantly.
        CLAUDE_BIN="$(command -v claude 2>/dev/null || true)"
        if [ -n "$CLAUDE_BIN" ]; then
            info "    claude CLI: ${CLAUDE_BIN}"
            BACKEND_STATE="claude CLI at ${CLAUDE_BIN}"
        else
            warn "--backend claudecode but the claude CLI is not on PATH"
            warn "install claude code first, or run theboringoffice with /backend opencode until then"
            BACKEND_STATE="claude CLI NOT on PATH (install claude code; office degrades loudly until then)"
        fi
    else
        BACKEND_STATE="default"
    fi
    seed_brain_backend
}

# ---------------------------------------------------------------- majdoor hook

# setup_majdoor_hook — opt-in (--majdoor-hook DIR): stamp the TheBoringMajdoor
# co-author trailer onto commits in ONE repo the member names. Pure delegation
# to scripts/install-majdoor-hook.sh (single source of truth for the hook body
# and the backup/hooks-dir rules): run from the checkout when install.sh runs
# in one, fetched from the raw URL into the temp workdir when curl-piped.
setup_majdoor_hook() {
    [ -n "$MAJDOOR_HOOK_DIR" ] || return 0
    stage "TheBoringMajdoor commit-msg hook"
    if [ "$DRY_RUN" -eq 1 ]; then
        info "  [dry-run] install the TheBoringMajdoor commit-msg attribution hook into: ${MAJDOOR_HOOK_DIR}"
        info "            (delegates to scripts/install-majdoor-hook.sh — checkout copy, else fetched raw)"
        MAJDOOR_STATE="would install into ${MAJDOOR_HOOK_DIR} (dry-run)"
        return 0
    fi
    if [ ! -d "$MAJDOOR_HOOK_DIR" ]; then
        warn "--majdoor-hook: ${MAJDOOR_HOOK_DIR} is not a directory — skipping (the office install itself is unaffected)"
        MAJDOOR_STATE="skipped (${MAJDOOR_HOOK_DIR} is not a directory)"
        return 0
    fi
    installer=""
    self_dir=$(CDPATH= cd "$(dirname "$0")" 2>/dev/null && pwd -P || true)
    if [ -n "$self_dir" ] && [ -f "${self_dir}/scripts/install-majdoor-hook.sh" ]; then
        installer="${self_dir}/scripts/install-majdoor-hook.sh"
    else
        # curl-pipe path: no checkout on disk — pull the installer AND the
        # hook body (the installer expects it as a sibling) into TMPWORK.
        fetch "${RAW_BASE}/scripts/install-majdoor-hook.sh" "${TMPWORK}/install-majdoor-hook.sh"
        fetch "${RAW_BASE}/scripts/majdoor-commit-msg-hook.sh" "${TMPWORK}/majdoor-commit-msg-hook.sh"
        installer="${TMPWORK}/install-majdoor-hook.sh"
    fi
    if sh "$installer" "$MAJDOOR_HOOK_DIR"; then
        MAJDOOR_STATE="installed into ${MAJDOOR_HOOK_DIR}"
    else
        warn "majdoor hook install FAILED — the office itself is fully installed; retry later with:"
        warn "  scripts/install-majdoor-hook.sh ${MAJDOOR_HOOK_DIR}"
        MAJDOOR_STATE="FAILED for ${MAJDOOR_HOOK_DIR} (see above)"
    fi
}

# ---------------------------------------------------------------- summary

box_hr() { info '+----------------------------------------------------------------------------'; }
box_row() { printf '| %s\n' "$*"; }

print_install_summary() {
    box_hr
    if [ "$DRY_RUN" -eq 1 ]; then
        box_row 'dry-run summary — NOTHING was actually changed'
    else
        box_row "theboringoffice installed"
    fi
    box_hr
    box_row "  binary  : ${PREFIX}/theboringoffice"
    box_row "  run it  : theboringoffice"
    if [ -n "$VERSION" ]; then
        box_row "  release : v${VERSION}"
    else
        box_row "  release : latest (unversioned fallback URL)"
    fi
    if [ "$PATH_HINT" -eq 1 ]; then
        box_row ""
        box_row "  PATH hint: ${PREFIX} is not on your PATH. Add:"
        box_row "      export PATH=\"${PREFIX}:\$PATH\""
        box_row "  to your ~/.zshrc or ~/.profile and restart your shell."
    fi
    box_row ""
    box_row "  agentmemory : ${AM_SERVICE_STATE}"
    box_row "  backend   : ${BACKEND} (${BACKEND_STATE})"
    if [ -n "$MAJDOOR_HOOK_DIR" ]; then
        box_row "  majdoor hook: ${MAJDOOR_STATE}"
    fi
    box_row "  check       : agentmemory status"
    box_hr
}

print_uninstall_summary() {
    box_hr
    if [ "$DRY_RUN" -eq 1 ]; then
        box_row 'dry-run uninstall summary — NOTHING was actually removed'
    else
        box_row 'theboringoffice uninstalled'
    fi
    box_hr
    box_row "  binary  : ${PREFIX}/theboringoffice removed"
    box_row "  service : ${PLIST_LABEL} / agentmemory.service removed"
    box_row "  kept    : ~/.agentmemory data, and the agentmemory npm package"
    box_hr
}

# ---------------------------------------------------------------- main

main() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --dry-run)          DRY_RUN=1 ;;
            --skip-agentmemory) SKIP_AGENTMEMORY=1 ;;
            --uninstall)        UNINSTALL=1 ;;
            --prefix)           [ $# -ge 2 ] || die "--prefix requires a DIR argument"
                                PREFIX=$2; shift ;;
            --prefix=*)         PREFIX=${1#--prefix=} ;;
            --backend)          [ $# -ge 2 ] || die "--backend requires a NAME argument"
                                BACKEND=$2; shift ;;
            --backend=*)        BACKEND=${1#--backend=} ;;
            --majdoor-hook)     [ $# -ge 2 ] || die "--majdoor-hook requires a DIR argument"
                                MAJDOOR_HOOK_DIR=$2; shift ;;
            --majdoor-hook=*)   MAJDOOR_HOOK_DIR=${1#--majdoor-hook=} ;;
            -h|--help)          usage; exit 0 ;;
            *)                  usage; die "unknown flag: $1" ;;
        esac
        shift
    done
    [ -n "$PREFIX" ] || PREFIX=""
    # fail fast — a mistyped transport should die HERE, not after downloads.
    [ "$BACKEND" = "opencode" ] || [ "$BACKEND" = "claudecode" ] \
        || die "--backend must be opencode|claudecode"

    print_banner

    if [ "$UNINSTALL" -eq 1 ]; then
        do_uninstall
        print_uninstall_summary
        exit 0
    fi

    detect_platform
    resolve_prefix_stage
    resolve_version
    download_assets
    verify_checksum
    install_binary
    setup_agentmemory
    setup_backend
    setup_majdoor_hook
    print_install_summary
}

resolve_prefix_stage() {
    stage "Choose install prefix"
    resolve_prefix
}

main "$@"
