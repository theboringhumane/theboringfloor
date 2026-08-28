// kitty_lanes.go — the NATIVE image lanes: on a terminal the detect
// layer (image_detect.go) proves speaks a real graphics protocol, a
// completed boss-turn image paints as the protocol's own escape frame
// instead of the ASCII half-block rows —
//
//	KittyLane → the kitty graphics protocol (kitty + ghostty): ONE
//	  placeholder strip per image, `ESC_G a=T,t=d,f=100,i=<hash8>,q=2;`
//	  + base64 PNG payload + `ESC\`, id = sha1(source bytes)[:8] hex;
//	ITermLane → iTerm2's OSC 1337 inline image (iTerm2, WezTerm, the VS
//	  Code terminal): `ESC]1337;File=inline=1;width=<cols>:height=<rows>;base64,<b64> BEL`,
//	  payload = the source bytes re-based (never re-encoded — iTerm2
//	  decodes png/jpeg/gif itself).
//
// Layout contract (the lane pick is INVISIBLE to the transcript grid):
// every lane reserves the SAME cell box the ASCII raster would occupy
// (rasterCells — 64×20 budget, aspect from the pixel side), the frame
// rides as ONE atomic row followed by the reservation rows, and a lane's
// bytes live OUTSIDE the stateful half-block view (chatMediaView.frame,
// never folded — a burst escape frame would corrupt the terminal).
//
// Safety contract: identical to the ASCII lane's — decodeMediaImage's
// caps + header-first dimension guard run before any payload escapes,
// corrupt bytes are an error (never a panic, never a half-written
// frame), and EVERY native-lane error falls back to the ASCII raster
// path before the caller's failed latch (a lane pick never blanks an
// image that ASCII could have painted).
package panels

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// LaneRender — ONE image's paint for a resolved lane: the native escape
// Frame (kitty/iterm) OR the ASCII half-block Rows, plus the transcript
// cell box either occupies (Cols × CellRows — identical arithmetic to
// the v1 raster, so the bubble layout never shifts with the lane).
type LaneRender struct {
	Lane     ImageLane // the lane that painted (ASCIILane after a fallback)
	Frame    string    // kitty strip / OSC 1337 — "" on the ASCII lane
	Rows     []string  // ASCII half-block rows — nil on the native lanes
	Cols     int       // reserved box, cells wide
	CellRows int       // reserved box, cells tall
}

// LaneRenderer is the native-transport seam: Render decodes src (the
// safe decodeMediaImage gate) and returns the lane's frame + reserved
// box. maxCols/maxRows take the same clampRasterBudget defaults+caps
// RasterFromBytes applies.
type LaneRenderer interface {
	Render(src []byte, maxCols, maxRows int) (LaneRender, error)
}

// KittyLaneRenderer — kitty graphics protocol (kitty's own marker,
// TERM_PROGRAM kitty|ghostty). Emits the placeholder strip; the payload
// is the image re-encoded PNG (f=100 is the protocol's only inline
// raster format — a PNG source re-encodes byte-identical under the
// stdlib encoder, a JPEG/GIF source converts).
type KittyLaneRenderer struct{}

// KittyImageID — the strip's i=<hash8>: the sha1 of the FULL source
// byte sequence, first 8 hex chars (4 bytes big-endian as a uint32, the
// protocol's 32-bit image-id space). Deterministic per payload, so the
// probe's hash-keyed store and this id agree forever.
func KittyImageID(src []byte) uint32 {
	sum := sha1.Sum(src)
	return binary.BigEndian.Uint32(sum[:4])
}

// KittyIDHash8 — the i= word exactly as the strip prints it (8 lowercase
// hex digits — sha1[:8] of the source bytes, never truncated from a
// longer prefix string).
func KittyIDHash8(id uint32) string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], id)
	return hex.EncodeToString(b[:])
}

// kittyFrameID — the office id a kitty placeholder strip carries (the
// i=<8-hex> word), plus whether the frame IS a kitty strip at all. The
// chat pane's frame-splice routing reads this at SetImageFrame time: a
// kitty strip with a parseable id rides the splice (the wrapper's
// emitted-set diff targets the id for a=d); an OSC 1337 frame carries
// NO id (iTerm2 has no image-delete escape either) and keeps the
// embedded-row behavior — the splice could never delete it, so a scroll
// would ghost it forever. A kitty frame whose id fails to parse reports
// id=0 (the caller falls back to embedding — never splice an image the
// diff cannot target).
func kittyFrameID(frame string) (uint32, bool) {
	if !strings.HasPrefix(frame, "\x1b_G") {
		return 0, false
	}
	const key = ",i="
	i := strings.Index(frame, key)
	if i < 0 || i+len(key)+8 > len(frame) {
		return 0, true
	}
	v, err := strconv.ParseUint(frame[i+len(key):i+len(key)+8], 16, 32)
	if err != nil {
		return 0, true
	}
	return uint32(v), true
}

// Render implements LaneRenderer.
func (KittyLaneRenderer) Render(src []byte, maxCols, maxRows int) (LaneRender, error) {
	maxCols, maxRows = clampRasterBudget(maxCols, maxRows)
	img, w, h, err := decodeMediaImage(src)
	if err != nil {
		return LaneRender{}, err
	}
	cols, cellRows := rasterCells(w, h, maxCols, maxRows)
	return LaneRender{
		Lane:  KittyLane,
		Frame: PlaceholderStrip(img, cols, cellRows, KittyImageID(src)),
		Cols:  cols, CellRows: cellRows,
	}, nil
}

// ITermLaneRenderer — iTerm2's OSC 1337 inline image (TERM_PROGRAM
// iTerm.app|iTerm, WezTerm's socket/name, the VS Code terminal). The
// payload is the SOURCE bytes base64'd verbatim (iTerm2 decodes the
// common image formats itself), sized into the reserved cell box via the
// width=<cols>:height=<rows> resize attributes, inline=1, BEL-terminated.
type ITermLaneRenderer struct{}

// ITermInlineFrame — the OSC 1337 wire shape for ONE image:
//
//	ESC ] 1337 ; File=inline=1;width=<cols>:height=<rows>;base64,<b64> BEL
func ITermInlineFrame(src []byte, cols, cellRows int) string {
	return fmt.Sprintf("\x1b]1337;File=inline=1;width=%d:height=%d;base64,%s\x07",
		cols, cellRows, base64.StdEncoding.EncodeToString(src))
}

// Render implements LaneRenderer.
func (ITermLaneRenderer) Render(src []byte, maxCols, maxRows int) (LaneRender, error) {
	maxCols, maxRows = clampRasterBudget(maxCols, maxRows)
	_, w, h, err := decodeMediaImage(src) // safety gate only — payload rides src verbatim
	if err != nil {
		return LaneRender{}, err
	}
	cols, cellRows := rasterCells(w, h, maxCols, maxRows)
	return LaneRender{
		Lane:  ITermLane,
		Frame: ITermInlineFrame(src, cols, cellRows),
		Cols:  cols, CellRows: cellRows,
	}, nil
}

// LaneRendererFor — the native renderer for a resolved lane; nil means
// "the ASCII lane" (the universal half-block rasterizer).
func LaneRendererFor(l ImageLane) LaneRenderer {
	switch l {
	case KittyLane:
		return KittyLaneRenderer{}
	case ITermLane:
		return ITermLaneRenderer{}
	default:
		return nil
	}
}

// ResolveImageLane — the probe's lane chain, posture × detection. The
// order is strict and total:
//
//	"ascii" → ASCIILane (the explicit pin always wins).
//	"auto"  → kitty → iterm → ascii: a detected KittyLane rides the kitty
//	  strip, a detected ITermLane rides OSC 1337, and EVERYTHING else
//	  (sixel — its renderer hasn't landed, none, plain ascii, tmux's
//	  conservative fold, unmapped terminals) keeps the ASCII lane, so
//	  the v1 paint never regresses.
//	anything else (incl. "off", which imagesOff gates upstream) → the
//	  conservative fold: ASCIILane.
func ResolveImageLane(mode string, detected ImageLane) ImageLane {
	switch mode {
	case "ascii":
		return ASCIILane
	case "auto":
		switch detected {
		case KittyLane:
			return KittyLane
		case ITermLane:
			return ITermLane
		default:
			return ASCIILane
		}
	default:
		return ASCIILane
	}
}

// RenderImageForLane — the probe's dispatcher: a native lane paints its
// escape frame; EVERY native-lane error falls back to the ASCII raster
// path (missing/broken bytes never panic and never blank an image ASCII
// could paint); only when ASCII ALSO fails does the error reach the
// caller, which degrades to the failed-chip latch exactly as v1.
func RenderImageForLane(l ImageLane, src []byte, maxCols, maxRows int) (LaneRender, error) {
	maxCols, maxRows = clampRasterBudget(maxCols, maxRows)
	if r := LaneRendererFor(l); r != nil {
		if lr, err := r.Render(src, maxCols, maxRows); err == nil {
			return lr, nil
		}
		// fall through: corrupt-for-this-lane bytes still deserve the
		// universal paint attempt before the chip degrades.
	}
	// The ASCII lane (and the native fallback): RasterFromBytes' exact
	// arithmetic, ONE decode shared by the box math and the paint.
	img, w, h, err := decodeMediaImage(src)
	if err != nil {
		return LaneRender{}, err
	}
	cols, cellRows := rasterCells(w, h, maxCols, maxRows)
	return LaneRender{
		Lane: ASCIILane,
		Rows: ByteToAnsiCells(img, cols, cellRows*2),
		Cols: cols, CellRows: cellRows,
	}, nil
}
