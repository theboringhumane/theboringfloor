// images.go — the inbound boss-turn image preview seam (ADDITIVE, the
// same ownership shape as the permission/question model-owned UI state):
//
// Wire contract (state.go owns the carrier): a completed boss turn can
// announce image file parts two ways, both deduped by MediaItem.Hash —
//
//	EvChatMedia (the SSE lane: message.part.updated sighted a file part
//	  on the primary session) — Msg.ID is the owning "bossmsg-"+messageID
//	  bubble identity, Media carries the payload (data-URL bytes inline);
//	EvChatBoss (the completion pin) — Msg.Meta carries the small
//	  "attach␟…␟hash" carrier (state.MediaMeta) and Event.Media re-
//	  announces the payloads (an older serve or a missed SSE frame still
//	  previews).
//
// Flow (the chatAttach lazy-probe idiom): applyMedia buffers each payload
// ONCE, keyed by hash (probe-once — imgProbed pins the rasterize cmd so
// a repeated pin never re-rasterizes), and fires ONE tea.Cmd per new
// payload (rasterizeCmd — a pure decode+render, the UI goroutine never
// decodes mid-frame). The landing (imageRasterMsg) pushes the paint into
// the chat panel (SetImageRaster for the ASCII lane, SetImageFrame for
// the native ones — block-cache friendly: the media revision folds into
// the carrier bubble's block key, so exactly ONE re-render happens per
// arrival) and bumps frameNonce.
//
// Posture (ui.images, the /images cycler):
//
//	off   → chips only ("🖼 name · WxH · mime"), no payload decode at all;
//	ascii → always the half-block truecolor raster (zero v1 regression);
//	auto  → the detect-layer chain, kitty → iterm → ascii (strict order):
//	        a kitty/ghostty terminal gets the kitty placeholder strip
//	        (ESC_G … ESC\), the iterm family (iTerm2 / WezTerm / VS Code)
//	        gets OSC 1337, and EVERYTHING else (tmux's conservative fold,
//	        sixel, none, unknowns) keeps the universal ASCII paint.
//
// The lane is resolved ONCE per payload at probe time (panels'
// DetectImageSupport memoized per boot on the Model — a cheap env read
// that never re-runs per frame) and a native render error falls back to
// the ASCII paint before the failed latch: a lane pick never blanks an
// image.
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// imageRasterMsg — the lazy rasterize landing (rasterizeCmd's result):
// the ASCII half-block rows painted, the native-lane escape frame
// painted, or failed=true (the chip drops to the dim
// "unsupported image" text). Keyed by msgID+hash — the same identity the
// probe latch pinned.
type imageRasterMsg struct {
	msgID    string
	hash     string
	rows     []string // ASCII lane paint
	frame    string   // native lane paint (kitty strip / OSC 1337)
	cellRows int      // native lane: the reserved cell-box height
	failed   bool
}

// applyMedia buffers inbound boss-turn image payloads (EvChatMedia SSE
// sightings and the EvChatBoss completion pin both land here — dedupe by
// hash) and probes the lazy render for each NEW one (ui.images posture
// permitting). Returns the probe cmds (nil when nothing new arrived,
// posture is off, or there is no chat panel yet).
func (m *Model) applyMedia(ev state.Event) tea.Cmd {
	if ev.Kind != state.EvChatMedia && ev.Kind != state.EvChatBoss {
		return nil
	}
	if len(ev.Media) == 0 {
		return nil
	}
	msgID := ev.Msg.ID
	if msgID == "" || m.imagesOff() || m.chat == nil {
		return nil
	}
	if m.imgProbed == nil {
		m.imgProbed = map[string]bool{}
	}
	var cmds []tea.Cmd
	for _, it := range ev.Media {
		if it.Hash == "" || it.URL == "" {
			continue // chip-only item (remote URL / undecodable) — never fetched, never probed
		}
		key := msgID + "|" + it.Hash
		if m.imgProbed[key] {
			continue
		}
		m.imgProbed[key] = true // probe-once: a repeated pin never re-rasterizes
		cmds = append(cmds, rasterizeCmd(msgID, it.Hash, it.URL, m.imageLane()))
		// The payload parse + decode happens INSIDE the cmd (async),
		// never on this goroutine; the panel renders the chip alone
		// until the landing repaints.
	}
	return tea.Batch(cmds...)
}

// imagesOff — the ui.images posture gate: "off" parks every probe
// (chips render from the Meta carrier alone). "auto" and "ascii" both
// probe (auto routes the detect chain, ascii pins the universal lane).
func (m *Model) imagesOff() bool {
	mode := "auto"
	if m.cfg != nil && m.cfg.UI.Images != "" {
		mode = m.cfg.UI.Images
	}
	switch mode {
	case "off":
		return true
	case "ascii", "auto":
		return false
	default:
		return true // unknown posture: the conservative fold
	}
}

// imageLane — the probe's render lane: the ui.images posture × the
// boot-memoized detect chain (kitty → iterm → ascii — strict, total,
// and ASCII for everything unmapped so v1 never regresses).
func (m *Model) imageLane() panels.ImageLane {
	mode := "auto"
	if m.cfg != nil && m.cfg.UI.Images != "" {
		mode = m.cfg.UI.Images
	}
	return panels.ResolveImageLane(mode, m.detectImageLane())
}

// detectImageLane — the terminal's image-lane read, memoized per boot
// (one env read per Model; probes and frames afterwards cost zero env
// traffic). Per-Model (not a package latch) so harnesses that stub the
// terminal env per drive — uishot's lane legs, the lane tests — each
// get a fresh honest read.
func (m *Model) detectImageLane() panels.ImageLane {
	if !m.imgLaneDetOK {
		m.imgLaneDet = panels.DetectImageSupport()
		m.imgLaneDetOK = true
	}
	return m.imgLaneDet
}

// onImagesLaneChanged — the /images cycler's ADDITIVE lane-flip hook,
// called after a posture write-through. Probe-once latencies are KEPT:
// landed previews never re-rasterize under the new posture (an ASCII
// paint on a kitty terminal stays valid paint), so the flip resolves
// lazily on the NEXT probe through imageLane(). The seam exists so a
// future transport that needs an eager re-probe has exactly one call
// site — clear imgProbed there, never at the cycler.
func (m *Model) onImagesLaneChanged() {
	// Intentionally effect-free today: the cycler's notice already
	// repaints, the detect memo survives (env outlives a posture flip),
	// and imgProbed stays pinned (the probe-once latency guarantee).
}

// rasterizeCmd — the lazy probe: decode the data URL + render OFF the
// UI goroutine (tea.Cmd semantics), landing back as imageRasterMsg. The
// lane was resolved by the caller (posture × detect chain); a native
// lane's render error falls back to the ASCII paint inside
// RenderImageForLane, and a double failure latches failed. The payload
// never crosses process or network boundaries (data URLs only — remote
// URLs were stripped at the wire gate).
func rasterizeCmd(msgID, hash, url string, lane panels.ImageLane) tea.Cmd {
	return func() tea.Msg {
		_, raw, err := state.ParseDataImageURL(url)
		if err != nil {
			return imageRasterMsg{msgID: msgID, hash: hash, failed: true}
		}
		lr, err := panels.RenderImageForLane(lane, raw, panels.RasterMaxCols, panels.RasterMaxRows)
		if err != nil || (lr.Frame == "" && len(lr.Rows) == 0) {
			return imageRasterMsg{msgID: msgID, hash: hash, failed: true}
		}
		return imageRasterMsg{
			msgID: msgID, hash: hash,
			rows: lr.Rows, frame: lr.Frame, cellRows: lr.CellRows,
		}
	}
}

// applyImageRaster — the landing: push the paint (the native frame, the
// ASCII rows, or the failed latch) into the chat panel and repaint. The
// panel folds its media revision into the carrier bubble's block key, so
// exactly one block re-renders per arrival; a failure flips just the
// chip text.
func (m *Model) applyImageRaster(msg imageRasterMsg) tea.Cmd {
	if m.chat == nil {
		return nil
	}
	if msg.failed {
		m.chat.SetImageFailed(msg.msgID, msg.hash)
		return nil
	}
	if msg.frame != "" {
		m.chat.SetImageFrame(msg.msgID, msg.hash, msg.frame, msg.cellRows)
		return nil
	}
	m.chat.SetImageRaster(msg.msgID, msg.hash, msg.rows)
	return nil
}
