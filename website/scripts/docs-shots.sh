#!/usr/bin/env bash
# =============================================================================
# docs-shots.sh — regenerate every documentation screenshot used by the
# theboringfloor website docs pages. Re-runnable and idempotent: every PNG in
# website/public/shots/docs/ is overwritten from a fresh deterministic capture.
#
# Pipeline per shot:
#   go run ./cmd/uishot <flags>          (seeded stub backend, 130x32, ~5s/run,
#    |                                      frames printed between "===== UI SHOT" markers)
#    | awk frame extraction              (1-indexed across marker pairs, 32 ANSI lines each)
#    | freeze <file.ansi> -c full -o <png>
#
# Toolchain (pinned 2026-08-26, darwin/arm64):
#   freeze version v0.2.2  (charmbracelet/freeze)
#     `freeze <file.ansi> -c full -o <out>.png` — DEFAULT font.size / line-height /
#     padding render a 130x32 uishot frame at EXACTLY 5086x2896, byte-matching the
#     canvas of website/public/shots/*.png. The file path preserves ANSI color
#     (verified on the pilot below) — no `freeze -x` workaround needed.
#   Resolution order: `freeze` on PATH, else $(go env GOPATH)/bin/freeze (the
#   `go install` location), else `go run github.com/charmbracelet/freeze@latest`.
#
#   PERF NOTE: freeze v0.2.2 is CPU-bound here — measured 42s for ONE 5086x2896
#   render (94% of one core). Renders therefore run in parallel batches of 6
#   (independent processes, disjoint outputs) — a full regen is ~6-7 minutes.
#
# Canvas contract: every PNG must come out 5086x2896. Extraction enforces
# exactly 32 ANSI lines per frame, and the summary re-measures every PNG.
#
# Pilot (2026-08-26): `go run ./cmd/uishot --tab agents | frame 1 > agents.ansi`
#   -> `freeze agents.ansi -c full -o agents.png` -> 5086x2896, full ANSI color.
#
# Frame() note: uishot frames OPEN with a (usually titled) line matching
# `^===== UI SHOT` and CLOSE with a plain `===== UI SHOT =====` line. A naive
# open/close toggle accumulates every frame before the wanted one for want>=2
# (their lines print before the wanted pair closes) — this implementation counts
# pairs and prints ONLY lines inside the wanted pair. Do not "simplify" it.
#
# Known-uishot quirk (2026-08-26): `--stop` currently exits 1 AFTER printing a
# valid frame ("expected exactly 1 AbortSessions call, got 0" — the wave-60
# async-abort pin does not observe the backend call on the deterministic drive).
# The frame content matches its markers (placeholders collapsed, tools ✗ aborted,
# "office › stopped by user", queue intact), so the PNG is rendered and the
# nonzero exit is reported in the summary's WARNINGS — fix uishot, not this file.
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="$REPO_ROOT/website/public/shots/docs"
RAW="$(mktemp -d /tmp/docs-shots.XXXXXX)"
trap 'rm -rf "$RAW"' EXIT
mkdir -p "$OUT_DIR"

if command -v freeze >/dev/null 2>&1; then
	FREEZE=(freeze)
elif [[ -x "$(go env GOPATH)/bin/freeze" ]]; then
	FREEZE=("$(go env GOPATH)/bin/freeze")
else
	echo "docs-shots: freeze not found — falling back to \`go run github.com/charmbracelet/freeze@latest\`" >&2
	FREEZE=(go run github.com/charmbracelet/freeze@latest)
fi

# frame <pair-index>: print only the 32 lines inside the wanted marker pair.
frame() {
	awk -v want="$1" '
		/^===== UI SHOT/{
			if (open) { open=0; if (n==want) exit; next }
			else { n++; open=1; next }
		}
		open && n==want'
}

# Static shot registry — png() runs in background subshells and cannot update
# parent-side arrays, so the summary iterates this list and reads the
# $RAW/ok-* or $RAW/skip-* status files each parallel png() leaves behind.
NAMES=(
	office-overview first-run-chat backend-claude chat-thinking
	work-threads thread-focus permission-modal question-modal concierge
	plan-gated plan-presented batch-dispatch board-sync stop-unwind
	terminal-tab git-tab layout-normal layout-compact layout-wide
	theme-dracula slash-popover model-picker
)

# capture <name> <uishot flags...>  — phase 1; never aborts on nonzero exit.
capture() {
	local name=$1
	shift
	local ec=0
	(cd "$REPO_ROOT" && go run ./cmd/uishot "$@") >"$RAW/$name.txt" 2>"$RAW/$name.err" || ec=$?
	if ((ec != 0)); then
		printf 'WARN  %-16s uishot exited %d — frame still validated on content (%s.err kept in workdir)\n' \
			"$name" "$ec" "$RAW/$name" >&2
		printf 'uishot exit %d' "$ec" >"$RAW/warn-$name"
	fi
}

# png <name> <frame#> <required-marker-fixed-string> <raw-capture-name>
# Extracts the frame, enforces 32 lines, greps the stripped frame for the
# marker, and renders freeze -c full. Any mismatch is SKIPped + reported,
# never silently rendered. Status lands in $RAW/ok-<name> or $RAW/skip-<name>
# (safe under parallel batches, where subshells cannot update parent arrays).
png() {
	local name=$1 want=$2 marker=$3 src=$4
	local ansi="$RAW/$name.ansi"
	frame "$want" <"$RAW/$src.txt" >"$ansi"
	if (($(wc -l <"$ansi") != 32)); then
		printf 'SKIP  %-18s frame %d of capture "%s" is not 32 lines\n' "$name" "$want" "$src" >&2
		printf 'frame %d of %s: not 32 lines' "$want" "$src" >"$RAW/skip-$name"
		return 0
	fi
	if ! sed $'s/\033\[[0-9;]*[a-zA-Z]//g' "$ansi" | grep -qF -- "$marker"; then
		printf 'SKIP  %-18s frame %d missing expected marker: %s\n' "$name" "$want" "$marker" >&2
		printf 'frame %d of %s: marker missing: %s' "$want" "$src" "$marker" >"$RAW/skip-$name"
		return 0
	fi
	"${FREEZE[@]}" "$ansi" -c full -o "$OUT_DIR/$name.png" >/dev/null
	printf 'ok' >"$RAW/ok-$name"
}

# -----------------------------------------------------------------------------
# Phase 1 — uishot captures (unique runs; shared runs captured once), batches
# of 5-6 parallel processes. The --claude run goes FIRST and ALONE: it compiles
# cmd/claudestub via `go build` at runtime and parallel `go run` siblings would
# race that build.
# -----------------------------------------------------------------------------
echo "docs-shots: phase 1 — capturing uishot frames (claude backend first, alone)…"
capture backend-claude --claude --planshot

capture office-overview --tab agents &
capture first-run-chat --tab chat &
capture git-tab --tab git &
capture theme-dracula --theme dracula --tab agents &
capture chat-thinking --think --think-stop mid &
wait

capture threads --threads &
capture threadfocus --threadfocus &
capture permission-modal --tab chat --at 2920 &
capture question-modal --ask-answer &
capture concierge --concierge &
wait

capture planshot --planshot &
capture batch-dispatch --batch &
capture boardsync --boardsync &
capture stop-unwind --stop &
capture terminal --terminal &
wait

capture layout --layout &
capture slashpop --slashpop &
capture modelshot --modelshot &
wait

# -----------------------------------------------------------------------------
# Phase 2 — extract (frame # 1-indexed) + marker check + freeze, in parallel
# batches of 6 (one freeze core each; see PERF NOTE above).
# Frame choices:
#   terminal-tab      = f2 of 4 ("PHASE 2 — ctrl+space CAPTURED: 'echo' typed,
#                        keys received: 4", capture hint row live) — the frame
#                        that best shows the terminal panel actually in use.
#   slash-popover     = f3 of 5 ("SLASH C — preview state 1 (↓ once): theme
#                        paper PAINTED LIVE", theme-preview popover open).
#   theme-dracula     marker is panel text: the dracula delta is ANSI SGR color
#                        only (frame text is byte-identical to office-overview).
#   batch-dispatch    marker is the composed flush summary in the transcript;
#                        the literal "[BATCH DISPATCH" envelope lives in the
#                        post-frame stub trace, not inside the frame.
# -----------------------------------------------------------------------------
echo "docs-shots: phase 2 — extracting + rendering 22 PNGs (freeze batches of 6)…"

png office-overview 1 'AGENTS' office-overview &                  # agents tab roster (boss/office/hr/tekton/skopos/dikastes rows)
png first-run-chat 1 'boss is typing' first-run-chat &            # chat tab: boss reply bubble + live typing row + concierge note
png backend-claude 1 'plan mode' backend-claude &                 # claude backend: plan mode active, chatter pinned, pane refuses
png chat-thinking 1 'thinking · 2 lines' chat-thinking &          # think-stream mid frame: expanded per-call thinking block
png work-threads 1 '✓ done' threads &                             # live collapsed thread beside a completed ✓ thread
png thread-focus 1 'esc · ctrl+f' threadfocus &                   # ctrl+f thread fullscreen: header + full body + back hint
wait
echo "docs-shots:   batch 1/4 done"

png permission-modal 1 'PERMISSION' permission-modal &            # t=2920ms: PERMISSION REQUIRED modal open over chat
png question-modal 1 'boss asks' question-modal &                 # question-answer modal open, parked status line
png concierge 1 'concierge' concierge &                           # boss busy: sends routed to concierge, notice once
png plan-gated 1 'plan mode' planshot &                           # plan mode badge + gated pane (floor kept, idle hint)
png plan-presented 2 'Goal' planshot &                            # plan-shaped reply presented: # Goal / # Steps in the pane
png batch-dispatch 1 'backlog dispatched' batch-dispatch &        # 3 queued items flushed as ONE composed batch send
wait
echo "docs-shots:   batch 2/4 done"

png board-sync 2 'board sync' boardsync &                         # after returns: "[office] board sync" note, oldest row flipped
png stop-unwind 1 'stopped by user' stop-unwind &                 # /stop: ✗ aborted tools, ✗ stopped thread, queue intact
png terminal-tab 2 'keys received: 4' terminal &                  # terminal panel captured: real typed echo round-trip
png git-tab 1 'untr' git-tab &                                    # git tab: mod/add/untr/del counters + changed files
png layout-normal 1 '1 chat' layout &                             # layout frame 1/3: NORMAL (sidebar 80, the default)
png layout-compact 2 'c   t   a' layout &                         # layout frame 2/3: compact (sidebar 30, short tab labels)
wait
echo "docs-shots:   batch 3/4 done"

png layout-wide 3 '2 terminal' layout &                           # layout frame 3/3: wide (sidebar 90)
png theme-dracula 1 'AGENTS' theme-dracula &                      # dracula palette over the agents tab (color-only delta)
png slash-popover 3 'theme preview' slashpop &                    # slash popover: /th filter → theme picker, live preview painted
png model-picker 1 'BOSS MODEL' modelshot &                       # /model picker open: 5-model listing, cursor row 1
wait
echo "docs-shots:   batch 4/4 done"

# -----------------------------------------------------------------------------
# Phase 3 — summary (+ uniform-canvas check via sips).
# -----------------------------------------------------------------------------
echo
echo "docs-shots: rendered -> $OUT_DIR"
skipped=0
for s in "${NAMES[@]}"; do
	if [[ -f "$RAW/ok-$s" ]]; then
		dims=$(
			sips -g pixelWidth -g pixelHeight "$OUT_DIR/$s.png" 2>/dev/null |
				awk '/pixelWidth/{w=$2} /pixelHeight/{h=$2} END{printf "%sx%s", w, h}'
		)
		printf '  OK   %-22s %s\n' "$s.png" "$dims"
	elif [[ -f "$RAW/skip-$s" ]]; then
		printf '  SKIP %-22s %s\n' "$s.png" "$(cat "$RAW/skip-$s")"
		skipped=$((skipped + 1))
	else
		printf '  SKIP %-22s internal: no status recorded\n' "$s.png"
		skipped=$((skipped + 1))
	fi
done
for f in "$RAW"/warn-*; do
	[[ -f $f ]] || continue
	printf '  WARN %-22s %s\n' "$(basename "$f" | sed 's/^warn-//')" "$(cat "$f")"
done

if ((skipped > 0)); then
	echo "docs-shots: $skipped shot(s) skipped" >&2
	exit 2
fi
