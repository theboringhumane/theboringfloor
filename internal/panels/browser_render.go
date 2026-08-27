// browser_render.go — the HTML → text-rows half of the browser tab.
//
// parseHTMLPage walks the DOM (golang.org/x/net/html) into a width-
// INDEPENDENT block model (Page.blocks); renderRows reflows that model
// into visual rows at each SetSize width. The extraction contract (spec
// §3):
//
//	<title>          → Page.Title + a title row at the body's head
//	h1..h6           → heading rows (the panel paints them bold)
//	p                → wrapped plain-text rows; inline <a> renders as
//	                   "anchor text [n]" — markdown-style, the [n] suffix
//	                   on the anchor's own text — with the resolved URLs
//	                   kept in Page.Links (stable appear order, deduped by
//	                   exact URL)
//	pre              → code rows, verbatim (never whitespace-collapsed)
//	ul/ol li         → "• " / "N. " bullet rows
//	table tr         → one row per tr, cells joined with " │ "
//	img              → "🖼 <alt-or-filename>" chip rows — image BYTES are
//	                   never fetched (v1 ignores the payload outright)
//	script/style/meta/comments/head chrome → stripped outright
//
// Page.Unsupported short-circuits everything: a non-HTML sniff renders as
// the single dim row "unsupported content type: <type>".
package panels

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Page — one fetched page: the location bar facts, the indexed link side
// map, and the width-independent block model the rows reflow from.
type Page struct {
	URL   string
	Title string
	Links []BrowserLink // [n] in appear order, deduped by exact URL
	// Unsupported is the sniffed content type when the payload was NOT
	// HTML ("" = parsed normally) — the pane renders the dim fallback row.
	Unsupported string

	blocks []block
}

// BrowserLink — one indexed link target: [n] ⇢ resolved URL + anchor text.
type BrowserLink struct {
	N    int    // 1-based (the rendered "[n]")
	URL  string // resolved absolute (http(s):// / file path)
	Text string // anchor text
}

// blockKind — the semantic row classes the panel styles by.
type blockKind int

const (
	blkTitle   blockKind = iota // the <title> line
	blkHeading                  // h1..h6 → bold
	blkPara                     // wrapped paragraph (may carry links)
	blkCode                     // pre — verbatim rows
	blkBullet                   // ul/ol item
	blkImage                    // 🖼 chip
	blkTable                    // one tr, " │ "-joined
	blkDim                      // dim fallback row (unsupported content type)
	blkSep                      // blank separator row between blocks
)

// block — one width-independent content unit; links indexes Page.Links.
// num > 0 on an ordered-list bullet (the "N. " marker replaces "• ").
type block struct {
	kind  blockKind
	text  string
	links []int
	num   int
}

// rowOut — one rendered visual row (raw text + the facts the panel's
// styling/cursor layers need).
type rowOut struct {
	text  string
	kind  blockKind
	links []int
}

// renderRows reflows the page's blocks into visual rows at wrap width w:
// paragraphs/bullets word-wrap, every other kind clips at the edge; a
// blank separator row parts two blocks of DIFFERENT kinds (bullet runs
// and table runs stay dense).
func (p *Page) renderRows(w int) []rowOut {
	if w < 10 {
		w = 10
	}
	if p.Unsupported != "" {
		return []rowOut{{text: "unsupported content type: " + p.Unsupported, kind: blkDim}}
	}
	var rows []rowOut
	for i, blk := range p.blocks {
		rows = append(rows, blk.rows(w)...)
		if i+1 < len(p.blocks) && (blk.kind != p.blocks[i+1].kind || !denseKind(blk.kind)) {
			rows = append(rows, rowOut{kind: blkSep})
		}
	}
	return rows
}

// denseKind — kinds whose consecutive siblings render back-to-back (no
// blank separator inside a list or a table).
func denseKind(k blockKind) bool {
	return k == blkBullet || k == blkTable || k == blkCode
}

// rows renders ONE block at wrap width w.
func (blk block) rows(w int) []rowOut {
	mk := func(text string) rowOut { return rowOut{text: text, kind: blk.kind, links: blk.links} }
	switch blk.kind {
	case blkTitle, blkHeading, blkImage, blkTable, blkDim:
		return []rowOut{mk(clipPlain(blk.text, w))}
	case blkCode:
		var outs []rowOut
		for _, ln := range strings.Split(blk.text, "\n") {
			outs = append(outs, mk(clipPlain(ln, w)))
		}
		if len(outs) == 0 {
			return []rowOut{mk("")}
		}
		return outs
	case blkBullet:
		marker := "• "
		if blk.num > 0 {
			marker = strconv.Itoa(blk.num) + ". "
		}
		return wrapBlock(mk, marker+blk.text, w)
	default: // blkPara
		return wrapBlock(mk, blk.text, w)
	}
}

// wrapBlock word-wraps one block's text at w cells, one rowOut per visual
// row (the block's link set rides every row of its run — the panel's
// focus highlight paints the whole run).
func wrapBlock(mk func(string) rowOut, text string, w int) []rowOut {
	var outs []rowOut
	for _, ln := range strings.Split(wrapPlain(text, w), "\n") {
		outs = append(outs, mk(ln))
	}
	if len(outs) == 0 {
		return []rowOut{mk("")}
	}
	return outs
}

// ---------------------------------------------------------------------------
// DOM → blocks
// ---------------------------------------------------------------------------

// parseHTMLPage parses the document and extracts the block model (title
// first, then the body's flow content in document order).
func parseHTMLPage(r io.Reader, pageURL string) (*Page, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	p := &Page{URL: pageURL}
	if t := findElement(doc, "title"); t != nil {
		p.Title = collapseWS(textContent(t))
	}
	if p.Title != "" {
		p.blocks = append(p.blocks, block{kind: blkTitle, text: p.Title})
	}
	body := findElement(doc, "body")
	if body == nil {
		body = doc
	}
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		p.walkBlock(c, pageURL)
	}
	return p, nil
}

// findElement — the first element with the given tag, depth-first.
func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if hit := findElement(c, tag); hit != nil {
			return hit
		}
	}
	return nil
}

// skipTags — stripped outright: their TEXT is noise (script bodies,
// styles, meta) and their children never render.
var skipTags = map[string]bool{
	"script": true, "style": true, "meta": true, "link": true,
	"noscript": true, "template": true, "head": true, "svg": true,
}

// containerTags — block-level elements the walker recurses INTO (their
// children carry the content; the wrapper itself emits nothing).
var containerTags = map[string]bool{
	"div": true, "section": true, "article": true, "main": true,
	"header": true, "footer": true, "figure": true, "figcaption": true,
	"blockquote": true, "dl": true, "nav": true, "aside": true,
}

// walkBlock folds one DOM node into the block list (document order).
func (p *Page) walkBlock(n *html.Node, pageURL string) {
	switch n.Type {
	case html.CommentNode, html.DoctypeNode:
		return
	case html.TextNode:
		if t := collapseWS(n.Data); t != "" {
			p.blocks = append(p.blocks, block{kind: blkPara, text: t})
		}
		return
	case html.ElementNode:
	default:
		return
	}
	tag := n.Data
	if skipTags[tag] {
		return
	}
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		text, links := p.inlineText(n, pageURL)
		if text != "" {
			p.blocks = append(p.blocks, block{kind: blkHeading, text: text, links: links})
		}
	case "p", "a":
		text, links := p.inlineText(n, pageURL)
		if text != "" {
			p.blocks = append(p.blocks, block{kind: blkPara, text: text, links: links})
		}
	case "pre":
		text := strings.Trim(textContent(n), "\n")
		if strings.TrimSpace(text) != "" {
			p.blocks = append(p.blocks, block{kind: blkCode, text: text})
		}
	case "ul", "ol":
		i := 0
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode || c.Data != "li" {
				continue
			}
			i++
			text, links := p.inlineText(c, pageURL)
			if text == "" {
				continue
			}
			b := block{kind: blkBullet, text: text, links: links}
			if tag == "ol" {
				b.num = i
			}
			p.blocks = append(p.blocks, b)
		}
	case "table":
		for _, tr := range findAll(n, "tr") {
			var cells []string
			for _, cell := range findAll(tr, "td", "th") {
				text, links := p.inlineText(cell, pageURL)
				_ = links // table cells keep their text only (v1)
				cells = append(cells, text)
			}
			if line := strings.Join(cells, " │ "); strings.TrimSpace(line) != "" {
				p.blocks = append(p.blocks, block{kind: blkTable, text: line})
			}
		}
	case "img":
		alt := attrOf(n, "alt")
		if alt == "" {
			src := attrOf(n, "src")
			alt = filepath.Base(src)
			if alt == "." || alt == "/" {
				alt = "image"
			}
		}
		p.blocks = append(p.blocks, block{kind: blkImage, text: "🖼 " + collapseWS(alt)})
	case "br", "hr":
		// no row worth a cell in v1
	default:
		if containerTags[tag] {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				p.walkBlock(c, pageURL)
			}
			return
		}
		// Unknown block-level element: treat its inline content as a
		// paragraph rather than dropping it (span/div-less markup).
		text, links := p.inlineText(n, pageURL)
		if text != "" {
			p.blocks = append(p.blocks, block{kind: blkPara, text: text, links: links})
		}
	}
}

// findAll — every descendant element matching any of the tags (document
// order, the node itself excluded).
func findAll(n *html.Node, tags ...string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				for _, t := range tags {
					if c.Data == t {
						out = append(out, c)
						break
					}
				}
			}
			walk(c)
		}
	}
	walk(n)
	return out
}

// attrOf — one element attribute ("" when absent).
func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent — the raw concatenated descendant text (entities already
// decoded by the parser); callers collapse or trim as needed.
func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		if cur.Type == html.TextNode {
			sb.WriteString(cur.Data)
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// collapseWS folds every whitespace run to one space and trims the ends.
func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// inlineText renders an element's inline content: text rides verbatim
// (whitespace-collapsed), <a> anchors render "text [n]" with the resolved
// URL indexed into Page.Links. Returns the text plus the link indexes in
// appear order.
func (p *Page) inlineText(n *html.Node, pageURL string) (string, []int) {
	var parts []string
	var links []int
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		switch cur.Type {
		case html.CommentNode:
			return
		case html.TextNode:
			if t := collapseWS(cur.Data); t != "" {
				parts = append(parts, t)
			}
			return
		case html.ElementNode:
			if skipTags[cur.Data] {
				return
			}
			if cur.Data == "a" {
				text := collapseWS(textContent(cur))
				href := resolveHref(pageURL, attrOf(cur, "href"))
				if text == "" {
					return
				}
				if href == "" {
					// anchor without a target (bare "#" etc): the text
					// renders plain — no dead [n].
					parts = append(parts, text)
					return
				}
				idx := p.linkIndex(href, text)
				links = append(links, idx)
				parts = append(parts, fmt.Sprintf("%s [%d]", text, idx+1))
				return
			}
			if cur.Data == "img" {
				// inline images ride the same chip contract as block ones
				alt := attrOf(cur, "alt")
				if alt == "" {
					alt = filepath.Base(attrOf(cur, "src"))
				}
				if alt != "" && alt != "." && alt != "/" {
					parts = append(parts, "🖼 "+collapseWS(alt))
				}
				return
			}
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(parts, " "), links
}

// linkIndex — the link side map: dedupe by EXACT resolved URL (the first
// occurrence keeps its index), stable appear order. Returns the 0-based
// index into Page.Links.
func (p *Page) linkIndex(resolvedURL, anchorText string) int {
	for i, l := range p.Links {
		if l.URL == resolvedURL {
			return i
		}
	}
	p.Links = append(p.Links, BrowserLink{N: len(p.Links) + 1, URL: resolvedURL, Text: anchorText})
	return len(p.Links) - 1
}

// resolveHref resolves one href against the page's URL: absolute URLs
// (http(s)/file) pass through; on an http(s) page, relative hrefs ride
// net/url's reference resolution; on a local page, they join the page's
// directory as filesystem paths. Pure anchors ("#…") resolve to "" (no
// target — the text renders plain).
func resolveHref(pageURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return ""
	}
	if strings.Contains(href, "://") {
		return href // absolute, scheme included
	}
	if strings.HasPrefix(pageURL, "http://") || strings.HasPrefix(pageURL, "https://") {
		base, err := url.Parse(pageURL)
		if err != nil {
			return ""
		}
		ref, err := url.Parse(href)
		if err != nil {
			return ""
		}
		return base.ResolveReference(ref).String()
	}
	// local page: filesystem resolution against the page's directory.
	local := strings.TrimPrefix(pageURL, "file://")
	if filepath.IsAbs(href) {
		return filepath.Clean(href)
	}
	return filepath.Join(filepath.Dir(local), href)
}
