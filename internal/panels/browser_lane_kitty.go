// browser_lane_kitty.go — the zenbu premium lane's KITTY PASSTHROUGH:
// the child's PTY stream is SPLIT in the lane's reader — kitty graphics
// APC escapes (ESC_G … ESC\, chunked m=1/m=0 transmissions, placements,
// deletes) are extracted into an image store, and EVERY OTHER byte flows
// to the scrollback + grid untouched, so the child's text chrome keeps
// painting exactly as before. The store's live images then reach the
// OUTER terminal through the FRAME-SPLICE seam (zenbu_frame.go — the
// Model publishes a per-Frame registry; the tea.WithOutput wrapper
// re-emits each image at its ABSOLUTE screen cell after every renderer
// flush), so kitty/ghostty composites the REAL rendered page (CSS,
// images) inside the left-pane browser slot. (The wave-80 design spliced
// the APC into the View string itself — bubbletea's cell renderer eats
// zero-width sequences, so that path never painted; the View now carries
// PURE TEXT and the images ride the wrapper.)
//
// THE WIRE (ground truth captured from terminal-browser v0.6.0's native
// engine on a real PTY): startup probes (`i=4207,a=q,t=d,f=24,s=1,v=1`
// + t=f/t=s fallbacks — queries the office deliberately never answers,
// the child tolerates the timeout), the alt screen + synchronized-update
// modes, cursor-home, then one FULL-WINDOW frame per repaint:
//
//	ESC_G a=T,f=32,o=z,s=1600,v=960,t=d,i=1,p=1,C=1,q=2,m=1;<b64> ESC\
//	ESC_G m=1;<b64> ESC\ … ESC_G m=0;<b64> ESC\
//
// (raw RGBA, zlib-compressed per o=z, chunked at ~4KB, ids reused,
// `_Ga=d,d=I,i=` / `_Ga=d,d=A,q=2` deletes on scroll/exit). The parser
// honors the whole kitty subset in play — f= (100 PNG / 32 RGBA / 24
// RGB), o=z, s=/v= pixel dims, c=/r= cell size, x=/y=/z= placement
// attrs, p= placement ids, C=1 cursor-stay, a=T/t/p/d/q.
//
// OFFICE-SIDE IDS are content-addressed (KittyImageID over the decoded
// payload — the chat lane's exact helper): the child's ids (i=1) are
// NEVER re-emitted, so a zenbu frame can never collide with a chat
// preview's id in the terminal's shared image namespace, and identical
// content re-transmits under the SAME office id (ghostty dedupes).
//
// RE-EMISSION CONTRACT: (a) the payload rides the child's OWN base64
// VERBATIM — decoded ONCE at commit for validation + the content hash,
// never re-encoded per frame (the assembled APC string is cached on the
// image; a repaint reuses it); (b) the image's cell origin is captured
// from the grid's cursor at the moment the transmission completed, which
// is exactly where the child's own cursor stood (C=1 frames never move
// it) — the frame wrapper CUPs to that pane-local offset PLUS the
// registry's absolute origin; (c) C=1 is forced on re-emission so the
// outer terminal's cursor never jumps past the image mid-row; (d)
// deletes ride ahead of placements (the wrapper's splice order), and on
// suspend/close they ALSO flush DIRECTLY to the terminal (the pane is
// gone — no frame will carry them) through the zenbuEmit seam (main.go
// wires it to the wrapper's DirectEmit — serialized with frame flushes;
// the sound bell's os.Stdout precedent: one self-contained APC, atomic
// at write granularity, invisible when no images live).
package panels

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/theboringhumane/theboringoffice/internal/term"
)

// zenbuMaxLiveImages bounds the store (the child reuses ids — a handful
// of live frames covers every observed pattern; overflow evicts oldest).
const zenbuMaxLiveImages = 8

// Stream caps (vars — the malformed-resilience tests shrink them).
var (
	// maxKittyAPCBody caps ONE APC's accumulated bytes (a full-window
	// compressed frame runs ~100KB–2MB; this is crash-runaway headroom).
	maxKittyAPCBody = 64 << 20
	// maxKittyChainB64 caps a chunk chain's joined base64 text.
	maxKittyChainB64 = 96 << 20
)

// zenbuEmit — the DIRECT-to-terminal seam for image deletes that can no
// longer ride a frame (suspend / close / office shutdown): the pane is
// not painted after those, so the a=d APCs write straight to the
// terminal (sound/player.go's bell-write precedent — one self-contained
// escape, atomic at syscall granularity against the renderer's
// one-write-per-frame). Production wires it to the frame wrapper's
// DirectEmit (cmd/theboringoffice — serialized with the renderer's
// flushes); the os.Stdout default only stands when no wrapper exists.
var zenbuEmit = func(s string) { _, _ = os.Stdout.WriteString(s) }

// SetZenbuEmit — the PRODUCTION wiring (cmd/theboringoffice's main): the
// frame wrapper's DirectEmit becomes the lane-lifecycle delete path for
// the process's whole run. Tests/shots keep the swap+restore seam below.
func SetZenbuEmit(fn func(string)) { zenbuEmit = fn }

// SetZenbuEmitForShot swaps the direct-emit seam for a shot/test harness
// (SetOpenRunnerForShot's exact precedent) and returns the restore.
func SetZenbuEmitForShot(fn func(string)) (restore func()) {
	old := zenbuEmit
	zenbuEmit = fn
	return func() { zenbuEmit = old }
}

// -------------------------------------------------------------------
// the APC parser
// -------------------------------------------------------------------

// kittyCmd — ONE complete kitty graphics APC (chunks already joined by
// the splitter): the action plus every key in play, numerics parsed,
// geometry strings kept verbatim for the passthrough re-emission.
type kittyCmd struct {
	action   byte   // a= value ('T','t','p','d','q',…); 0 = chunk continuation
	format   string // f= verbatim ("100" PNG, "32" RGBA, "24" RGB)
	medium   byte   // t= medium ('d' direct, 'f' file, 's' shm, 't' temp); 0 absent
	okey     string // o= ("z" = zlib-compressed payload)
	id       uint32 // i=
	hasID    bool
	number   uint32 // I=
	hasNum   bool
	place    uint32 // p=
	hasPlace bool
	more     int // m= (1 = more chunks, 0 = final — the spec default when absent)
	cols     string
	rows     string
	x, y, z  string
	pw, ph   string // s=, v= pixel dims (required for f=32/24)
	ckey     string // C= (1 = cursor does not move after display)
	ukey     string // U= (1 = unicode-placeholder display — NOT supported)
	delWhat  string // d= delete selector (i/I/a/A/p/P/q/Q/n/N/z/Z/c/C)
	b64      string // the payload's base64 text (chunks joined; "" allowed)
}

// key — the store's lookup: i= wins, I= (image number) is the fallback.
func (c kittyCmd) key() (uint32, bool) {
	if c.hasID {
		return c.id, true
	}
	if c.hasNum {
		return c.number, true
	}
	return 0, false
}

// kittyValueOK — passthrough values are integers or single letters on
// the real wire; anything else would be a re-injection risk into the
// office's own APC (the whole command drops as malformed instead).
func kittyValueOK(v string) bool {
	if len(v) == 0 || len(v) > 16 {
		return false
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// parseKittyAPC parses the bytes between ESC_G and the ST/BEL terminator.
// Never panics: every malformation is an error (the splitter drops it,
// logs once, and keeps streaming).
func parseKittyAPC(body []byte) (kittyCmd, error) {
	var c kittyCmd // c.more defaults to 0 (the spec: absent m = final chunk)
	s := string(body)
	ctrl, payload := s, ""
	if i := strings.IndexByte(s, ';'); i >= 0 {
		ctrl, payload = s[:i], s[i+1:]
	}
	c.b64 = payload
	if ctrl == "" && payload == "" {
		return c, fmt.Errorf("empty APC")
	}
	for _, tok := range strings.Split(ctrl, ",") {
		if tok == "" {
			continue
		}
		k, v, found := strings.Cut(tok, "=")
		if !found || len(k) != 1 {
			return c, fmt.Errorf("bad key token %q", tok)
		}
		switch k[0] {
		case 'a':
			if len(v) != 1 {
				return c, fmt.Errorf("bad action %q", v)
			}
			c.action = v[0]
		case 'f':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad format %q", v)
			}
			c.format = v
		case 't':
			if len(v) != 1 {
				return c, fmt.Errorf("bad medium %q", v)
			}
			c.medium = v[0]
		case 'o':
			if v != "z" {
				return c, fmt.Errorf("bad compression %q", v)
			}
			c.okey = v
		case 'i':
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return c, fmt.Errorf("bad image id %q", v)
			}
			c.id, c.hasID = uint32(n), true
		case 'I':
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return c, fmt.Errorf("bad image number %q", v)
			}
			c.number, c.hasNum = uint32(n), true
		case 'p':
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return c, fmt.Errorf("bad placement id %q", v)
			}
			c.place, c.hasPlace = uint32(n), true
		case 'm':
			n, err := strconv.Atoi(v)
			if err != nil || (n != 0 && n != 1) {
				return c, fmt.Errorf("bad chunk flag %q", v)
			}
			c.more = n
		case 'q': // response suppression — honored, never re-emitted verbatim
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad quiet %q", v)
			}
		case 'c':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad cols %q", v)
			}
			c.cols = v
		case 'r':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad rows %q", v)
			}
			c.rows = v
		case 'x':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad x %q", v)
			}
			c.x = v
		case 'y':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad y %q", v)
			}
			c.y = v
		case 'z':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad z %q", v)
			}
			c.z = v
		case 's':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad s %q", v)
			}
			c.pw = v
		case 'v':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad v %q", v)
			}
			c.ph = v
		case 'C':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad C %q", v)
			}
			c.ckey = v
		case 'U':
			if !kittyValueOK(v) {
				return c, fmt.Errorf("bad U %q", v)
			}
			c.ukey = v
		case 'd':
			if len(v) != 1 {
				return c, fmt.Errorf("bad delete selector %q", v)
			}
			c.delWhat = v
		default:
			// forward-compat: unknown keys are ignored, never fatal.
		}
	}
	return c, nil
}

// -------------------------------------------------------------------
// the image store
// -------------------------------------------------------------------

// zenbuImage — one live transmission: the child's base64 payload kept
// VERBATIM (re-emission never re-encodes), the office-side content id,
// and the placement the last a=T/a=p established.
type zenbuImage struct {
	childID  uint32
	officeID uint32
	format   string
	okey     string
	pw, ph   string
	b64      string

	placed  bool
	place   uint32 // p= placement id (0 = kitty's default placement)
	ox, oy  int    // pane-local cell origin (the grid cursor at commit)
	cols    string
	rows    string
	x, y, z string

	frame string // the cached re-emission ("" = rebuild)
	seq   uint64
}

// emitLocked — the cached office-side transmit+display frame: the
// child's payload + geometry keys verbatim under the OFFICE content id,
// C=1 forced (the outer cursor must never jump past the image mid-row),
// q=2 forced (the office never answers). Builds ONCE per payload+attrs
// (placement mutations clear frame) — a repaint reuses the string.
func (im *zenbuImage) emitLocked() string {
	if im.frame != "" {
		return im.frame
	}
	var b strings.Builder
	b.Grow(len(im.b64) + 128)
	fmt.Fprintf(&b, "\x1b_Ga=T,t=d,q=2,C=1,i=%08x", im.officeID)
	if im.format != "" {
		b.WriteString(",f=" + im.format)
	}
	if im.okey != "" {
		b.WriteString(",o=" + im.okey)
	}
	if im.pw != "" {
		b.WriteString(",s=" + im.pw)
	}
	if im.ph != "" {
		b.WriteString(",v=" + im.ph)
	}
	if im.cols != "" {
		b.WriteString(",c=" + im.cols)
	}
	if im.rows != "" {
		b.WriteString(",r=" + im.rows)
	}
	if im.x != "" {
		b.WriteString(",x=" + im.x)
	}
	if im.y != "" {
		b.WriteString(",y=" + im.y)
	}
	if im.z != "" {
		b.WriteString(",z=" + im.z)
	}
	if im.place != 0 {
		fmt.Fprintf(&b, ",p=%d", im.place)
	}
	b.WriteString(";")
	b.WriteString(im.b64)
	b.WriteString("\x1b\\")
	im.frame = b.String()
	return im.frame
}

// kittyDeleteFrame — the office-side delete for one content id: d=I
// deletes the image AND frees its data (the terminal's memory never
// retains a frame past the lane's lifecycle).
func kittyDeleteFrame(officeID uint32) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%08x,q=2;\x1b\\", officeID)
}

// zenbuPlacement — the frame-splice registry's unit: ONE live image's
// office content id + cached emit frame + its pane-local cell origin
// (frames materialized under the store lock; the registry/wrapper never
// touches the store).
type zenbuPlacement struct {
	officeID uint32
	ox, oy   int
	z        int
	seq      uint64
	frame    string
}

// zenbuImageStore — the lane's live kitty state: keyed by the child's
// image id (i=, I= fallback), latest-transmission-wins, bounded (oldest
// evicts, its office delete queued), deletes honored for the id/all/
// placement/z-index selectors. Safe for concurrent use (the PTY reader
// applies, the frame publish drains).
type zenbuImageStore struct {
	mu       sync.Mutex
	images   map[uint32]*zenbuImage
	order    []uint32 // insertion order (the eviction ledger)
	seq      uint64
	pending  []uint32 // queued office-side delete ids
	pendSeen map[uint32]bool
	drops    int
	note     string // the FIRST malformed reason (log-once)
}

func newZenbuImageStore() *zenbuImageStore {
	return &zenbuImageStore{images: map[uint32]*zenbuImage{}, pendSeen: map[uint32]bool{}}
}

// malformed — the log-once discipline: the first reason pins, every
// later one only bumps the counter (never a panic, never a flood).
func (s *zenbuImageStore) malformed(format string, args ...any) {
	s.drops++
	if s.note == "" {
		s.note = fmt.Sprintf(format, args...)
	}
}

// queueDeleteLocked — dedupe until drained (the same office id never
// deletes twice in one publish interval).
func (s *zenbuImageStore) queueDeleteLocked(officeID uint32) {
	if s.pendSeen[officeID] {
		return
	}
	s.pendSeen[officeID] = true
	s.pending = append(s.pending, officeID)
}

// apply — one COMPLETE command (chunk chains already joined) + the
// pane-local cell origin (the grid's cursor at completion).
func (s *zenbuImageStore) apply(cmd kittyCmd, ox, oy int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch cmd.action {
	case 'T', 't':
		s.applyTransmitLocked(cmd, ox, oy)
	case 'p':
		s.applyPlaceLocked(cmd, ox, oy)
	case 'd':
		s.applyDeleteLocked(cmd)
	case 'q', 0:
		// queries (the child's startup probes) and bare chunk markers:
		// nothing to store; the office deliberately never answers.
	default:
		// a=f / a=r / future actions: not in play — silently ignored.
	}
}

// applyTransmitLocked — a=T / a=t: decode+validate ONCE (the content
// hash + the malformed gate), latest-wins replace, the replaced image's
// office delete queued (the terminal must not composite stale frames).
func (s *zenbuImageStore) applyTransmitLocked(cmd kittyCmd, ox, oy int) {
	if cmd.ukey != "" {
		s.malformed("unicode-placeholder frames (U=%s) unsupported", cmd.ukey)
		return
	}
	if cmd.medium != 0 && cmd.medium != 'd' {
		s.malformed("unsupported transmission medium t=%c", cmd.medium)
		return
	}
	if cmd.b64 == "" {
		s.malformed("transmit without payload")
		return
	}
	key, ok := cmd.key()
	if !ok {
		s.malformed("transmit without image id")
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(cmd.b64)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(cmd.b64, "="))
		if err != nil {
			s.malformed("undecodable payload (%v)", err)
			return
		}
	}
	officeID := KittyImageID(decoded)
	if old := s.images[key]; old != nil && old.placed && old.officeID != officeID {
		s.queueDeleteLocked(old.officeID)
	}
	if _, exists := s.images[key]; !exists {
		for len(s.images) >= zenbuMaxLiveImages && len(s.order) > 0 {
			victim := s.order[0]
			s.order = s.order[1:]
			if im := s.images[victim]; im != nil {
				if im.placed {
					s.queueDeleteLocked(im.officeID)
				}
				delete(s.images, victim)
			}
		}
		s.order = append(s.order, key)
	}
	s.seq++
	s.images[key] = &zenbuImage{
		childID: key, officeID: officeID,
		format: cmd.format, okey: cmd.okey, pw: cmd.pw, ph: cmd.ph,
		b64:    cmd.b64,
		placed: cmd.action == 'T', place: cmd.place,
		ox: ox, oy: oy,
		cols: cmd.cols, rows: cmd.rows, x: cmd.x, y: cmd.y, z: cmd.z,
		seq: s.seq,
	}
}

// applyPlaceLocked — a=p: (re)place an already-transmitted image; the
// placement command's geometry overrides the transmission's.
func (s *zenbuImageStore) applyPlaceLocked(cmd kittyCmd, ox, oy int) {
	key, ok := cmd.key()
	if !ok {
		s.malformed("placement without image id")
		return
	}
	im := s.images[key]
	if im == nil {
		s.malformed("placement for unknown image id %d", key)
		return
	}
	im.placed = true
	im.ox, im.oy = ox, oy
	if cmd.hasPlace {
		im.place = cmd.place
	}
	if cmd.cols != "" {
		im.cols = cmd.cols
	}
	if cmd.rows != "" {
		im.rows = cmd.rows
	}
	if cmd.x != "" {
		im.x = cmd.x
	}
	if cmd.y != "" {
		im.y = cmd.y
	}
	if cmd.z != "" {
		im.z = cmd.z
	}
	s.seq++
	im.seq = s.seq
	im.frame = "" // geometry changed — the cached emit rebuilds
}

// applyDeleteLocked — a=d: the child's delete selectors mapped to the
// office id(s) they currently name.
func (s *zenbuImageStore) applyDeleteLocked(cmd kittyCmd) {
	what := cmd.delWhat
	if what == "" {
		what = "i" // kitty's default: delete by id
	}
	switch what {
	case "i", "I", "n", "N":
		key, ok := cmd.key()
		if !ok {
			s.malformed("delete without image id")
			return
		}
		if im := s.images[key]; im != nil {
			if im.placed {
				s.queueDeleteLocked(im.officeID)
			}
			delete(s.images, key)
		}
	case "a", "A":
		for k, im := range s.images {
			if im.placed {
				s.queueDeleteLocked(im.officeID)
			}
			delete(s.images, k)
		}
		s.order = s.order[:0]
	case "p", "P", "q", "Q":
		key, ok := cmd.key()
		if !ok || !cmd.hasPlace {
			s.malformed("placement delete without id/placement")
			return
		}
		if im := s.images[key]; im != nil && im.place == cmd.place {
			if im.placed {
				s.queueDeleteLocked(im.officeID)
			}
			delete(s.images, key)
		}
	case "z", "Z":
		for k, im := range s.images {
			if im.z == cmd.z {
				if im.placed {
					s.queueDeleteLocked(im.officeID)
				}
				delete(s.images, k)
			}
		}
	default:
		// d=c (image under the terminal's cursor) and friends: the
		// office's cursor model doesn't track placements that way; the
		// next frame/delete-all converges the terminal regardless.
	}
}

// placements — the frame-splice registry's image list: every placed
// image with its cached emit frame materialized (built under the lock,
// so the cached string is shared across repaints), sorted paint-order
// (origin row, origin col, z, then insertion seq — later frames
// composite on top).
func (s *zenbuImageStore) placements() []zenbuPlacement {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]zenbuPlacement, 0, len(s.images))
	for _, im := range s.images {
		if !im.placed {
			continue
		}
		z := 0
		if im.z != "" {
			if n, err := strconv.Atoi(im.z); err == nil {
				z = n
			}
		}
		out = append(out, zenbuPlacement{officeID: im.officeID, ox: im.ox, oy: im.oy, z: z, seq: im.seq, frame: im.emitLocked()})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].oy != out[j].oy {
			return out[i].oy < out[j].oy
		}
		if out[i].ox != out[j].ox {
			return out[i].ox < out[j].ox
		}
		if out[i].z != out[j].z {
			return out[i].z < out[j].z
		}
		return out[i].seq < out[j].seq
	})
	return out
}

// drainPendingIDs — the queued office-side delete ids (in-queue order),
// clearing the queue so a re-queue after a re-placement works. The frame
// publish drains per render (bounded); dropAll drains at Close.
func (s *zenbuImageStore) drainPendingIDs() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) == 0 {
		return nil
	}
	out := s.pending
	s.pending = nil
	s.pendSeen = map[uint32]bool{}
	return out
}

// retirePlacements — the pane resized: every live placement's geometry
// is stale — queue the deletes (the child repaints after SIGWINCH and
// re-places fresh) and un-place, payloads kept (the re-transmit of the
// same content reuses the same office id: delete-then-add converges).
func (s *zenbuImageStore) retirePlacements() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, im := range s.images {
		if im.placed {
			s.queueDeleteLocked(im.officeID)
			im.placed = false
		}
	}
}

// dropAll — the Close path: queue deletes for every live placement and
// return the concatenated delete frames for the direct-emit seam (""
// when the terminal holds nothing of ours — no stray bytes then).
func (s *zenbuImageStore) dropAll() string {
	s.mu.Lock()
	for _, im := range s.images {
		if im.placed {
			s.queueDeleteLocked(im.officeID)
			im.placed = false
		}
	}
	s.mu.Unlock()
	ids := s.drainPendingIDs()
	var b strings.Builder
	for _, id := range ids {
		b.WriteString(kittyDeleteFrame(id))
	}
	return b.String()
}

// dropStats — the malformed log-once surface (drops total + the FIRST
// reason), for the resilience tests + the harness's evidence line.
func (s *zenbuImageStore) dropStats() (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.drops, s.note
}

// -------------------------------------------------------------------
// the stream splitter (the lane reader's io.Writer)
// -------------------------------------------------------------------

// kittyStream — an io.Writer sitting between the PTY master and the
// scrollback+grid: complete kitty graphics APCs are extracted to the
// store (chunked m=1 chains joined until m=0), EVERY other byte flows
// downstream untouched. Split reads, malformed sequences (drop, log-once,
// never panic), and interleaved text are all first-class.
type kittyStream struct {
	dst   io.Writer
	grid  *term.Grid
	store *zenbuImageStore

	pending []byte // undispatched tail (APC body in APC mode, ESC tail in text mode)
	inAPC   bool

	chain    *kittyCmd       // the open chunk chain's first-chunk command
	chainB64 strings.Builder // the chain's joined payload text
}

func newKittyStream(dst io.Writer, grid *term.Grid, store *zenbuImageStore) *kittyStream {
	return &kittyStream{dst: dst, grid: grid, store: store}
}

// Write implements io.Writer (the reader loop's only entry): never
// errors (io.Copy keeps draining), O(len(p)) in text mode.
func (k *kittyStream) Write(p []byte) (int, error) {
	n := len(p)
	k.pending = append(k.pending, p...)
	for {
		if k.inAPC {
			if !k.scanAPC() {
				break
			}
		} else {
			if !k.scanText() {
				break
			}
		}
	}
	return n, nil
}

// scanText — text mode: flush plain runs downstream, hold trailing
// partial ESC introducers, switch to APC mode on ESC_G. Returns false
// when more bytes are needed.
func (k *kittyStream) scanText() bool {
	i := strings.IndexByte(string(k.pending), 0x1b)
	if i < 0 {
		_, _ = k.dst.Write(k.pending)
		k.pending = k.pending[:0]
		return false
	}
	// disambiguation needs ESC + 2 more bytes.
	if i+2 >= len(k.pending) {
		if i+1 < len(k.pending) && k.pending[i+1] != '_' {
			// ESC + a non-APC byte: downstream's business — flush through
			// the ESC and keep scanning after it.
			_, _ = k.dst.Write(k.pending[:i+1])
			k.pending = k.pending[i+1:]
			return true
		}
		// trailing ESC (or ESC _): could grow into ESC_G — hold.
		_, _ = k.dst.Write(k.pending[:i])
		k.pending = k.pending[i:]
		return false
	}
	if k.pending[i+1] == '_' && k.pending[i+2] == 'G' {
		_, _ = k.dst.Write(k.pending[:i])
		k.pending = k.pending[i+3:]
		k.inAPC = true
		return true
	}
	// not a graphics APC: the ESC rides downstream; keep scanning.
	_, _ = k.dst.Write(k.pending[:i+1])
	k.pending = k.pending[i+1:]
	return true
}

// scanAPC — APC mode: accumulate until the ST (ESC\) or BEL terminator,
// commit the completed command, resync on a stray ESC (malformed). The
// body is capped (maxKittyAPCBody) so a runaway unterminated stream
// drops instead of wedging the lane.
func (k *kittyStream) scanAPC() bool {
	for j := 0; j < len(k.pending); j++ {
		switch k.pending[j] {
		case 0x07:
			k.commit(k.pending[:j])
			k.pending = k.pending[j+1:]
			k.inAPC = false
			return true
		case 0x1b:
			if j+1 >= len(k.pending) {
				return k.capAPC() // the ST's second byte is still coming
			}
			if k.pending[j+1] == '\\' {
				k.commit(k.pending[:j])
				k.pending = k.pending[j+2:]
				k.inAPC = false
				return true
			}
			// a stray ESC inside an APC: payloads are base64 (never ESC)
			// — drop the body and resync AT the ESC (it may open a fresh
			// sequence).
			k.store.malformed("APC aborted by stray ESC (resync)")
			k.pending = k.pending[j:]
			k.inAPC = false
			return true
		}
	}
	return k.capAPC()
}

// capAPC — the unterminated-runaway bound: false while the body can
// still grow, true (after dropping) when it busted the cap.
func (k *kittyStream) capAPC() bool {
	if len(k.pending) <= maxKittyAPCBody {
		return false
	}
	k.store.malformed("unterminated APC over the %d-byte cap (dropped)", maxKittyAPCBody)
	k.pending = k.pending[:0]
	k.inAPC = false
	return true
}

// commit — one completed APC body: parse, join chunk chains (m=1 …
// m=0), and apply completed commands against the store with the pane's
// CURRENT cell origin (the grid's cursor — every preceding byte has
// already painted, so it is exactly the child's cursor at emission).
func (k *kittyStream) commit(body []byte) {
	cmd, err := parseKittyAPC(body)
	if err != nil {
		k.store.malformed("bad APC: %v", err)
		return
	}
	if cmd.action != 0 {
		// a full command: any open chain is broken (chunks must ride
		// back-to-back APCs) — drop it, process the new command.
		if k.chain != nil {
			k.store.malformed("chunk chain interrupted by a=%c", cmd.action)
			k.chain = nil
			k.chainB64.Reset()
		}
		if cmd.more == 1 {
			c := cmd // the loop's cmd copy escapes to the chain
			k.chain = &c
			k.chainB64.WriteString(cmd.b64)
			return
		}
		k.applyNow(cmd)
		return
	}
	// an actionless APC: ONLY valid as a chain continuation.
	if k.chain == nil {
		k.store.malformed("chunk without an open chain")
		return
	}
	k.chainB64.WriteString(cmd.b64)
	if k.chainB64.Len() > maxKittyChainB64 {
		k.store.malformed("chunk chain over the %d-byte cap (dropped)", maxKittyChainB64)
		k.chain = nil
		k.chainB64.Reset()
		return
	}
	if cmd.more != 0 {
		return // more chunks coming
	}
	fin := *k.chain
	fin.b64 = k.chainB64.String()
	k.chain = nil
	k.chainB64.Reset()
	k.applyNow(fin)
}

// applyNow — the completed command lands on the store at the grid's
// current cursor (the pane-local placement origin).
func (k *kittyStream) applyNow(cmd kittyCmd) {
	ox, oy := 0, 0
	if k.grid != nil {
		ox, oy = k.grid.Cursor()
	}
	k.store.apply(cmd, ox, oy)
}
