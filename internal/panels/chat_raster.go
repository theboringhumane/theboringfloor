// chat_raster.go — the ASCII image lane: truecolor half-block (`▀`)
// rasterization of decoded image bytes into terminal-ready ANSI strings,
// pure Go, stdlib only (image/png, image/jpeg, image/gif — Decode picks by
// content sniff, so a .png-named JPEG still reads).
//
// Cell model: ONE terminal cell paints TWO VERTICAL pixels — the top pixel
// as the FOREGROUND color of `▀`, the bottom as its BACKGROUND — so N
// pixel rows render as N/2 character rows and the pixel aspect stays
// honest on a 2:1 terminal cell (ESC[38;2;r;g;bm + ESC[48;2;r;g;bm —
// the same truecolor SGR the terminal emulator's grid parses).
//
// Downsampling is a hand-rolled box average (stdlib image/draw has no
// downscaler — NearestNeighborhood is its "fast" scaler, Copy is not a
// resampler): each destination pixel averages the WHOLE source region it
// covers (rounded per-axis), so a red|blue checkerboard blends to real
// 127-purple instead of aliasing.
//
// Safety contract (decode never allocates unbounded memory and never
// panics — every malformed input lands as an error, which the caller
// degrades into a dim chip row):
//   - source bytes capped at rasterMaxSrcBytes (8 MiB — the repo's
//     atMaxSize-scale payload contract);
//   - DecodeConfig runs BEFORE Decode: dimensions > 10000×10000 are
//     rejected from the header alone (no pixel allocation);
//   - output is capped at RasterMaxCols × RasterMaxRows cells.
package panels

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // register gif (first frame flattens)
	_ "image/jpeg" // register jpeg
	"image/png"    // register png + re-encode for the kitty strip
	"strings"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	// rasterMaxDim bounds EITHER decoded axis (checked header-first via
	// DecodeConfig, before any pixel buffer is allocated).
	rasterMaxDim = 10000
	// RasterMaxCols / RasterMaxRows — the transcript's raster budget per
	// image (cells). 64×20 keeps one preview under a typical sidebar page.
	RasterMaxCols = 64
	RasterMaxRows = 20
)

// halfBlockGlyph is the one-cell two-pixel paint: fg = top, bg = bottom.
const halfBlockGlyph = "▀"

// RasterFromBytes decodes image bytes (png/jpeg/gif, content-sniffed) and
// renders them as truecolor half-block ANSI rows, ≤ maxCols cells wide
// and ≤ maxRows character rows tall (aspect-preserved from the pixel
// side: a cell paints a 1×2 pixel column, so source 8×8 → 8 cols × 4
// rows). Errors for: oversized bytes, undecodable content, and
// header-detected giant dimensions (no allocation ever happens for
// those) — the caller degrades every one of them into a chip row.
func RasterFromBytes(src []byte, maxCols, maxRows int) ([]string, error) {
	maxCols, maxRows = clampRasterBudget(maxCols, maxRows)
	img, w, h, err := decodeMediaImage(src)
	if err != nil {
		return nil, err
	}
	cols, cellRows := rasterCells(w, h, maxCols, maxRows)
	return ByteToAnsiCells(img, cols, cellRows*2), nil
}

// clampRasterBudget — the transcript's raster budget defaults + caps,
// shared by every lane (the native lanes reserve the SAME cell box the
// ASCII lane would paint, so a lane pick never shifts the layout).
func clampRasterBudget(maxCols, maxRows int) (int, int) {
	if maxCols <= 0 {
		maxCols = RasterMaxCols
	}
	if maxRows <= 0 {
		maxRows = RasterMaxRows
	}
	if maxCols > RasterMaxCols {
		maxCols = RasterMaxCols
	}
	if maxRows > RasterMaxRows {
		maxRows = RasterMaxRows
	}
	return maxCols, maxRows
}

// decodeMediaImage — the shared safe-decode gate every lane passes
// through: the byte caps, the header-first dimension check, then the
// decode itself. Same contract (and same errors) RasterFromBytes has
// always carried: oversized bytes, undecodable content, and
// header-detected giant dimensions all land as errors, which the caller
// degrades into a dim chip row.
func decodeMediaImage(src []byte) (img image.Image, w, h int, err error) {
	if len(src) > state.MediaMaxPayloadBytes {
		return nil, 0, 0, fmt.Errorf("image payload over the 8 MiB cap (%d bytes)", len(src))
	}
	if len(src) == 0 {
		return nil, 0, 0, fmt.Errorf("empty image payload")
	}
	// Header-first bounds check: DecodeConfig reads the headers ONLY —
	// a 20000×20000 claim costs a few bytes of parse, never an image
	// allocation (image.Decode would happily build the pixel buffer).
	cfg, _, err := image.DecodeConfig(bytes.NewReader(src))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("unsupported image: %v", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > rasterMaxDim || cfg.Height > rasterMaxDim {
		return nil, 0, 0, fmt.Errorf("image %dx%d outside the 1..%d dimension budget", cfg.Width, cfg.Height, rasterMaxDim)
	}
	img, _, err = image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("unsupported image: %v", err)
	}
	return img, cfg.Width, cfg.Height, nil
}

// rasterCells — the transcript cell box for ONE image, shared by ALL
// lanes. Aspect from the PIXEL side: cols ≤ width, pixel rows = 2 × cell
// rows (the ASCII cell paints a 1×2 pixel column; the native lanes
// reserve the same box so the bubble layout is lane-independent). The
// row count is what a terminal spends, so rows clamp first, then width.
func rasterCells(w, h, maxCols, maxRows int) (cols, cellRows int) {
	cols = w
	if cols > maxCols {
		cols = maxCols
	}
	pxRows := h * cols / w
	if pxRows < 1 {
		pxRows = 1
	}
	cellRows = (pxRows + 1) / 2
	if cellRows > maxRows {
		cellRows = maxRows
	}
	return cols, cellRows
}

// PlaceholderStrip — ONE kitty graphics-protocol transmit-and-display
// frame (kitty terminals; ghostty speaks the same protocol):
//
//	ESC_G a=T,t=d,f=100,i=<id8>,q=2;<base64(png)> ESC\
//
// a=T transmits AND displays at the cursor, t=d inlines the payload
// (direct — no temp-file medium), f=100 marks the payload PNG (the
// protocol's only inline raster format — non-PNG sources re-encode,
// deterministic under the stdlib encoder), i= is the image id rendered
// as 8 lowercase hex digits (the caller's sha1[:8] of the source bytes —
// full-length hash, never a truncated-prefix guess), q=2 hushes kitty's
// response escape. cellW/cellH document the cell box the CALLER reserves
// for the paint (the transcript row ledger lives in renderMediaRows) —
// the strip itself stays exactly the pinned wire shape. Base64 rides in
// its standard 4-octet groups, encoded once per image; the probe's
// hash-keyed store caches the finished strip, so a re-render never
// re-encodes.
func PlaceholderStrip(img image.Image, cellW, cellH int, id uint32) string {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img) // a decoded stdlib image always re-encodes
	return fmt.Sprintf("\x1b_Ga=T,t=d,f=100,i=%08x,q=2;", id) +
		base64.StdEncoding.EncodeToString(buf.Bytes()) + "\x1b\\"
}

// ByteToAnsiCells downsamples img to (cols × cellH) destination pixels
// (cellH is the PIXEL-row count — 2 per terminal row) with an exact box
// average, then paints each pair of pixel rows as one half-block ANSI
// row: fg = top pixel, bg = bottom pixel, palette emitted as
// "38;2;r;g;b" / "48;2;r;g;b" SGR, batched per run so a uniform stretch
// costs one escape instead of two per cell. Each returned row is a
// self-contained styled string (roster-render pipelines treat rows
// atomically — a row never relies on a previous row's open escape).
func ByteToAnsiCells(img image.Image, cols, cellH int) []string {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw < 1 || sh < 1 || cols < 1 || cellH < 1 {
		return nil
	}
	// Box-average pass: dst pixel (x,y) averages the half-open source
	// rectangle [ x*sw/cols, (x+1)*sw/cols ) × [ y*sh/cellH, (y+1)*sh/cellH ).
	// cols ≤ sw and cellH ≤ sh hold by construction (the caller clamps
	// both), so every cell's rectangle is ≥1 pixel, the cells partition
	// the source exactly, and no pixel is ever double-counted. Alpha is
	// averaged in and flattened onto black, the terminal's universal
	// backdrop.
	px := make([][3]uint8, cols*cellH)
	for y := 0; y < cellH; y++ {
		sy0 := y * sh / cellH
		sy1 := (y + 1) * sh / cellH
		if sy1 >= sh {
			sy1 = sh
		}
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < cols; x++ {
			sx0 := x * sw / cols
			sx1 := (x + 1) * sw / cols
			if sx1 >= sw {
				sx1 = sw
			}
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var rr, gg, bb, aa, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					r, g, bl, a := img.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					rr += uint64(r)
					gg += uint64(g)
					bb += uint64(bl)
					aa += uint64(a)
					n++
				}
			}
			// 16-bit channel averages → flatten onto black → 8-bit.
			r16 := flatten(rr/n, aa/n)
			g16 := flatten(gg/n, aa/n)
			b16 := flatten(bb/n, aa/n)
			px[y*cols+x] = [3]uint8{uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8)}
		}
	}
	rows := make([]string, 0, (cellH+1)/2)
	for y := 0; y < cellH; y += 2 {
		var row strings.Builder
		var top, bot [3]uint8 // last-emitted palette (the run latch)
		have := false
		for x := 0; x < cols; x++ {
			t := px[y*cols+x]
			bm := t // odd source heights flatten the bottom twin into itself
			if y+1 < cellH {
				bm = px[(y+1)*cols+x]
			}
			if !have || t != top || bm != bot {
				fmt.Fprintf(&row, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", t[0], t[1], t[2], bm[0], bm[1], bm[2])
				top, bot, have = t, bm, true
			}
			row.WriteString(halfBlockGlyph)
		}
		row.WriteString("\x1b[m")
		rows = append(rows, row.String())
	}
	return rows
}

// flatten composites one 16-bit channel over black at 16-bit alpha.
func flatten(c, a uint64) uint64 {
	if a >= 0xffff {
		return c
	}
	return c * a / 0xffff
}
