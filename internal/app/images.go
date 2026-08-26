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
// payload (rasterizeCmd — a pure decode+box-downsample, the UI goroutine
// never decodes mid-frame). The landing (imageRasterMsg) pushes rows into
// the chat panel (SetImageRaster — block-cache friendly: the media
// revision folds into the carrier bubble's block key, so exactly ONE
// re-render happens per arrival) and bumps frameNonce.
//
// Posture (ui.images, the /images cycler):
//
//	off   → chips only ("🖼 name · WxH · mime"), no payload decode at all;
//	ascii → always the half-block truecolor raster;
//	auto  → the detect-layer chain (kitty → iterm → ascii); v1 renders
//	        the ASCII lane for every native pick (the lanes are the
//	        upgrade path), NoneLane (TERM dumb) folds to chips.
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// imageRasterMsg — the lazy rasterize landing (rasterizeCmd's result):
// rows painted, or failed=true (the chip drops to the dim
// "unsupported image" text). Keyed by msgID+hash — the same identity the
// probe latch pinned.
type imageRasterMsg struct {
	msgID  string
	hash   string
	rows   []string
	failed bool
}

// applyMedia buffers inbound boss-turn image payloads (EvChatMedia SSE
// sightings and the EvChatBoss completion pin both land here — dedupe by
// hash) and probes the lazy rasterize for each NEW one (ui.images posture
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
		cmds = append(cmds, rasterizeCmd(msgID, it.Hash, it.URL))
		// The payload parse + decode happens INSIDE the cmd (async),
		// never on this goroutine; the panel renders the chip alone
		// until the landing repaints.
	}
	return tea.Batch(cmds...)
}

// imagesOff — the ui.images posture gate: "off" parks every probe
// (chips render from the Meta carrier alone). "auto" folds to ASCII on a
// byte-dumb TERM (NoneLane), renders ASCII through the detect chain
// otherwise (kitty/iterm/sixel horizons keep the v1 fallback).
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

// rasterizeCmd — the lazy probe: decode the data URL + rasterize OFF the
// UI goroutine (tea.Cmd semantics), landing back as imageRasterMsg. The
// payload never crosses process or network boundaries (data URLs only —
// remote URLs were stripped at the wire gate).
func rasterizeCmd(msgID, hash, url string) tea.Cmd {
	return func() tea.Msg {
		_, raw, err := state.ParseDataImageURL(url)
		if err != nil {
			return imageRasterMsg{msgID: msgID, hash: hash, failed: true}
		}
		rows, err := panels.RasterFromBytes(raw, panels.RasterMaxCols, panels.RasterMaxRows)
		if err != nil || len(rows) == 0 {
			return imageRasterMsg{msgID: msgID, hash: hash, failed: true}
		}
		return imageRasterMsg{msgID: msgID, hash: hash, rows: rows}
	}
}

// applyImageRaster — the landing: push the rows (or the failed latch)
// into the chat panel and repaint. The panel folds its media revision
// into the carrier bubble's block key, so exactly one block re-renders
// per arrival; a failure flips just the chip text.
func (m *Model) applyImageRaster(msg imageRasterMsg) tea.Cmd {
	if m.chat == nil {
		return nil
	}
	if msg.failed {
		m.chat.SetImageFailed(msg.msgID, msg.hash)
		return nil
	}
	m.chat.SetImageRaster(msg.msgID, msg.hash, msg.rows)
	return nil
}
