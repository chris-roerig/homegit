package web

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// LinkResolver rewrites a URL found in markdown. Relative URLs are mapped to
// repo browser URLs; image is true for image sources. Implementations return
// "" to drop an unsafe URL.
type LinkResolver func(url string, image bool) string

// RenderMarkdown converts a practical subset of GitHub-flavored markdown to
// HTML: headings, paragraphs, emphasis, code, links, images, lists (nested,
// task items), blockquotes, tables, rules, and a small allowlist of inline
// HTML tags. All other text is escaped. Output is safe to embed in a page.
func RenderMarkdown(src string, resolve LinkResolver) string {
	if resolve == nil {
		resolve = func(u string, _ bool) string { return safeURL(u) }
	}
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src = strings.TrimPrefix(src, "\ufeff")
	r := &mdRenderer{resolve: resolve, slugs: map[string]int{}}
	return r.blocks(strings.Split(src, "\n"), false)
}

type mdRenderer struct {
	resolve LinkResolver
	slugs   map[string]int
}

var (
	reATX       = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)
	reFence     = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})[ \\t]*([^`]*)$")
	reSetext1   = regexp.MustCompile(`^ {0,3}=+[ \t]*$`)
	reSetext2   = regexp.MustCompile(`^ {0,3}-+[ \t]*$`)
	reListItem  = regexp.MustCompile(`^( {0,3})([-*+]|\d{1,9}[.)])([ \t]+|$)(.*)$`)
	reTableSep  = regexp.MustCompile(`^ {0,3}\|?[ \t]*:?-+:?[ \t]*(\|[ \t]*:?-+:?[ \t]*)*\|?[ \t]*$`)
	reHTMLBlock = regexp.MustCompile(`(?i)^\s*</?(p|div|details|summary|center|h[1-6]|hr|br|table|thead|tbody|tr|td|th|picture|source|ul|ol|li|blockquote|pre)\b`)
	reTag       = regexp.MustCompile(`^<(/?)([a-zA-Z][a-zA-Z0-9]*)((?:\s+[^<>]*?)?)\s*(/?)>`)
	reAttr      = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_-]*)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	reAutolink  = regexp.MustCompile(`^<(https?://[^\s<>]+|mailto:[^\s<>]+)>`)
	reDim       = regexp.MustCompile(`^\d{1,4}%?$`)
	reBareURL   = regexp.MustCompile(`^https?://[^\s<>]+`)
)

func indentOf(s string) int {
	n := 0
	for _, c := range s {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 4 - n%4
		default:
			return n
		}
	}
	return n
}

// dedent removes up to n columns of leading whitespace.
func dedent(s string, n int) string {
	cols, i := 0, 0
	for i < len(s) && cols < n {
		switch s[i] {
		case ' ':
			cols++
		case '\t':
			cols += 4 - cols%4
		default:
			return s[i:]
		}
		i++
	}
	return s[i:]
}

// isHR reports whether a line is a thematic break (---, ***, ___).
func isHR(l string) bool {
	if indentOf(l) >= 4 {
		return false
	}
	t := strings.NewReplacer(" ", "", "\t", "").Replace(l)
	if len(t) < 3 || strings.IndexByte("-*_", t[0]) < 0 {
		return false
	}
	return strings.Trim(t, string(t[0])) == ""
}

func isBlank(s string) bool { return strings.TrimSpace(s) == "" }

func (r *mdRenderer) blocks(lines []string, tight bool) string {
	var out strings.Builder
	i := 0
	for i < len(lines) {
		line := lines[i]
		if isBlank(line) {
			i++
			continue
		}

		// Fenced code.
		if m := reFence.FindStringSubmatch(line); m != nil {
			fence, info := m[2], strings.TrimSpace(m[3])
			open := len(m[1])
			var body []string
			i++
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if strings.HasPrefix(t, fence) && strings.Trim(t, string(fence[0])) == "" {
					i++
					break
				}
				body = append(body, dedent(lines[i], open))
				i++
			}
			out.WriteString("<pre><code")
			if f := strings.Fields(info); len(f) > 0 {
				out.WriteString(` class="language-` + html.EscapeString(f[0]) + `"`)
			}
			out.WriteString(">")
			out.WriteString(html.EscapeString(strings.Join(body, "\n")))
			if len(body) > 0 {
				out.WriteString("\n")
			}
			out.WriteString("</code></pre>\n")
			continue
		}

		// ATX heading.
		if m := reATX.FindStringSubmatch(line); m != nil {
			out.WriteString(r.heading(len(m[1]), m[2]))
			i++
			continue
		}

		// Horizontal rule.
		if isHR(line) {
			out.WriteString("<hr>\n")
			i++
			continue
		}

		// HTML comment: dropped.
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "<!--") {
			for i < len(lines) && !strings.Contains(lines[i], "-->") {
				i++
			}
			i++
			continue
		}

		// Indented code.
		if indentOf(line) >= 4 {
			var body []string
			for i < len(lines) && (isBlank(lines[i]) || indentOf(lines[i]) >= 4) {
				body = append(body, dedent(lines[i], 4))
				i++
			}
			for len(body) > 0 && isBlank(body[len(body)-1]) {
				body = body[:len(body)-1]
			}
			out.WriteString("<pre><code>" + html.EscapeString(strings.Join(body, "\n")) + "\n</code></pre>\n")
			continue
		}

		// Blockquote.
		if strings.HasPrefix(strings.TrimLeft(line, " "), ">") && indentOf(line) < 4 {
			var body []string
			for i < len(lines) && !isBlank(lines[i]) {
				t := strings.TrimLeft(lines[i], " ")
				if strings.HasPrefix(t, ">") {
					t = strings.TrimPrefix(t, ">")
					t = strings.TrimPrefix(t, " ")
				} else if len(body) == 0 || isBlockStart(lines[i]) {
					break
				}
				body = append(body, t)
				i++
			}
			out.WriteString("<blockquote>\n" + r.blocks(body, false) + "</blockquote>\n")
			continue
		}

		// List.
		if m := reListItem.FindStringSubmatch(line); m != nil {
			var s string
			s, i = r.list(lines, i)
			out.WriteString(s)
			continue
		}

		// Table.
		if i+1 < len(lines) && strings.Contains(line, "|") && reTableSep.MatchString(lines[i+1]) && strings.Contains(lines[i+1], "-") {
			header := splitRow(line)
			aligns := splitRow(lines[i+1])
			if len(header) == len(aligns) {
				var s string
				s, i = r.table(lines, i, header, aligns)
				out.WriteString(s)
				continue
			}
		}

		// HTML block line (allowlisted tags): emit inline-processed, no <p>.
		if reHTMLBlock.MatchString(line) {
			out.WriteString(r.inline(strings.TrimSpace(line)) + "\n")
			i++
			continue
		}

		// Paragraph (or setext heading).
		var para []string
		level := 0
		for i < len(lines) {
			l := lines[i]
			if isBlank(l) {
				break
			}
			if len(para) > 0 {
				if reSetext1.MatchString(l) {
					level = 1
					i++
					break
				}
				if reSetext2.MatchString(l) {
					level = 2
					i++
					break
				}
				if isBlockStart(l) || reHTMLBlock.MatchString(l) {
					break
				}
			}
			para = append(para, strings.TrimLeft(l, " \t"))
			i++
		}
		text := strings.TrimRight(strings.Join(para, "\n"), " \t")
		switch {
		case level > 0:
			out.WriteString(r.heading(level, text))
		case tight:
			out.WriteString(r.inline(text) + "\n")
		default:
			out.WriteString("<p>" + r.inline(text) + "</p>\n")
		}
	}
	return out.String()
}

// isBlockStart reports whether a line begins a block that can interrupt a
// paragraph.
func isBlockStart(l string) bool {
	if reATX.MatchString(l) || reFence.MatchString(l) || isHR(l) {
		return true
	}
	if strings.HasPrefix(strings.TrimLeft(l, " "), ">") {
		return true
	}
	if m := reListItem.FindStringSubmatch(l); m != nil && strings.TrimSpace(m[4]) != "" {
		// Ordered lists only interrupt a paragraph when starting at 1.
		if m[2][0] >= '0' && m[2][0] <= '9' {
			return strings.TrimRight(m[2], ".)") == "1"
		}
		return true
	}
	return false
}

func (r *mdRenderer) heading(level int, text string) string {
	text = strings.TrimSpace(text)
	slug := slugify(stripInline(text))
	if slug == "" {
		slug = "section"
	}
	n := r.slugs[slug]
	r.slugs[slug] = n + 1
	if n > 0 {
		slug = fmt.Sprintf("%s-%d", slug, n)
	}
	return fmt.Sprintf("<h%d id=\"%s\">%s</h%d>\n", level, html.EscapeString(slug), r.inline(text), level)
}

var (
	reStripLink = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	reSlugBad   = regexp.MustCompile(`[^\p{L}\p{N}\- _]`)
)

func stripInline(s string) string {
	s = reStripLink.ReplaceAllString(s, "$1")
	return strings.NewReplacer("`", "", "*", "", "~~", "").Replace(s)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = reSlugBad.ReplaceAllString(s, "")
	return strings.ReplaceAll(s, " ", "-")
}

// list parses a list starting at lines[start] and returns its HTML and the
// index of the first unconsumed line.
func (r *mdRenderer) list(lines []string, start int) (string, int) {
	first := reListItem.FindStringSubmatch(lines[start])
	ordered := first[2][0] >= '0' && first[2][0] <= '9'
	startNum := 1
	if ordered {
		startNum, _ = strconv.Atoi(strings.TrimRight(first[2], ".)"))
	}

	var items [][]string
	loose := false
	i := start

	for i < len(lines) {
		m := reListItem.FindStringSubmatch(lines[i])
		if m == nil || (m[2][0] >= '0' && m[2][0] <= '9') != ordered {
			break
		}
		// Content column: marker width plus following spaces (1-4).
		pad := len(m[3])
		if pad > 4 || strings.TrimSpace(m[4]) == "" {
			pad = 1
		}
		contentCol := len(m[1]) + len(m[2]) + pad
		cur := []string{m[4]}
		i++
		inFence := reFence.MatchString(cur[0])
		for i < len(lines) {
			l := lines[i]
			if isBlank(l) {
				// Look ahead to decide whether the item continues.
				j := i
				for j < len(lines) && isBlank(lines[j]) {
					j++
				}
				if j >= len(lines) {
					i = j
					break
				}
				if indentOf(lines[j]) >= contentCol {
					for k := i; k < j; k++ {
						cur = append(cur, "")
					}
					i = j
					if !inFence {
						loose = true
					}
					continue
				}
				if sm := reListItem.FindStringSubmatch(lines[j]); sm != nil && len(sm[1]) < contentCol && (sm[2][0] >= '0' && sm[2][0] <= '9') == ordered {
					loose = true
					i = j
				} else {
					i = j
					items = append(items, cur)
					return r.renderList(items, ordered, startNum, loose), i
				}
				break
			}
			if indentOf(l) >= contentCol {
				d := dedent(l, contentCol)
				if reFence.MatchString(d) {
					inFence = !inFence
				}
				cur = append(cur, d)
				i++
				continue
			}
			if sm := reListItem.FindStringSubmatch(l); sm != nil && !inFence {
				break // sibling (or end of this list type; checked at loop top)
			}
			if !inFence && !isBlockStart(l) && !isBlank(cur[len(cur)-1]) {
				cur = append(cur, strings.TrimSpace(l)) // lazy continuation
				i++
				continue
			}
			break
		}
		items = append(items, cur)
	}

	return r.renderList(items, ordered, startNum, loose), i
}

var reTask = regexp.MustCompile(`^\[([ xX])\][ \t]+`)

func (r *mdRenderer) renderList(items [][]string, ordered bool, startNum int, loose bool) string {
	var b strings.Builder
	tag := "ul"
	if ordered {
		tag = "ol"
	}
	hasTask := false
	for _, it := range items {
		if len(it) > 0 && reTask.MatchString(it[0]) {
			hasTask = true
		}
	}
	b.WriteString("<" + tag)
	if ordered && startNum != 1 {
		b.WriteString(fmt.Sprintf(` start="%d"`, startNum))
	}
	if hasTask {
		b.WriteString(` class="task-list"`)
	}
	b.WriteString(">\n")
	for _, it := range items {
		b.WriteString("<li>")
		if len(it) > 0 {
			if m := reTask.FindStringSubmatch(it[0]); m != nil {
				checked := ""
				if m[1] != " " {
					checked = " checked"
				}
				b.WriteString(`<input type="checkbox" disabled` + checked + "> ")
				it = append([]string{it[0][len(m[0]):]}, it[1:]...)
			}
		}
		body := r.blocks(it, !loose)
		if loose {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimRight(body, "\n"))
		if loose {
			b.WriteString("\n")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</" + tag + ">\n")
	return b.String()
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = strings.TrimSuffix(line, "|")
	}
	var cells []string
	var cur strings.Builder
	inCode := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\\' && i+1 < len(line) && line[i+1] == '|':
			cur.WriteByte('|')
			i++
		case c == '`':
			inCode = !inCode
			cur.WriteByte(c)
		case c == '|' && !inCode:
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}

func (r *mdRenderer) table(lines []string, i int, header, aligns []string) (string, int) {
	al := make([]string, len(aligns))
	for k, a := range aligns {
		l, rt := strings.HasPrefix(a, ":"), strings.HasSuffix(a, ":")
		switch {
		case l && rt:
			al[k] = ` style="text-align:center"`
		case rt:
			al[k] = ` style="text-align:right"`
		case l:
			al[k] = ` style="text-align:left"`
		}
	}
	var b strings.Builder
	b.WriteString("<table>\n<thead><tr>")
	for k, h := range header {
		b.WriteString("<th" + al[k] + ">" + r.inline(h) + "</th>")
	}
	b.WriteString("</tr></thead>\n<tbody>\n")
	i += 2
	for i < len(lines) && !isBlank(lines[i]) && strings.Contains(lines[i], "|") {
		cells := splitRow(lines[i])
		b.WriteString("<tr>")
		for k := range header {
			c := ""
			if k < len(cells) {
				c = cells[k]
			}
			b.WriteString("<td" + al[k] + ">" + r.inline(c) + "</td>")
		}
		b.WriteString("</tr>\n")
		i++
	}
	b.WriteString("</tbody>\n</table>\n")
	return b.String(), i
}

// ---- inline ----

func isPunct(c byte) bool {
	return strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' }

func isWordByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func (r *mdRenderer) inline(s string) string {
	var b strings.Builder
	n := len(s)
	for i := 0; i < n; {
		c := s[i]
		switch c {
		case '\\':
			if i+1 < n && s[i+1] == '\n' {
				b.WriteString("<br>\n")
				i += 2
				continue
			}
			if i+1 < n && isPunct(s[i+1]) {
				b.WriteString(html.EscapeString(string(s[i+1])))
				i += 2
				continue
			}
			b.WriteByte('\\')
			i++
		case '`':
			j := i
			for j < n && s[j] == '`' {
				j++
			}
			run := s[i:j]
			if end := findRun(s, j, run); end >= 0 {
				code := strings.ReplaceAll(s[j:end], "\n", " ")
				if len(code) > 2 && code[0] == ' ' && code[len(code)-1] == ' ' && strings.TrimSpace(code) != "" {
					code = code[1 : len(code)-1]
				}
				b.WriteString("<code>" + html.EscapeString(code) + "</code>")
				i = end + len(run)
			} else {
				b.WriteString(run)
				i = j
			}
		case '!', '[':
			if out, adv, ok := r.link(s, i); ok {
				b.WriteString(out)
				i += adv
				continue
			}
			b.WriteByte(c)
			i++
		case '<':
			if out, adv, ok := r.angle(s, i); ok {
				b.WriteString(out)
				i += adv
				continue
			}
			b.WriteString("&lt;")
			i++
		case '*', '_', '~':
			if out, adv, ok := r.emphasis(s, i); ok {
				b.WriteString(out)
				i += adv
				continue
			}
			b.WriteByte(c)
			i++
		case 'h':
			if (i == 0 || !isWordByte(s[i-1])) && strings.HasPrefix(s[i:], "http") {
				if m := reBareURL.FindString(s[i:]); m != "" {
					u := strings.TrimRight(m, ".,;:!?'\"*_~")
					for strings.HasSuffix(u, ")") && strings.Count(u, "(") < strings.Count(u, ")") {
						u = u[:len(u)-1]
					}
					if href := safeURL(u); href != "" {
						b.WriteString(`<a href="` + html.EscapeString(href) + `">` + html.EscapeString(u) + "</a>")
						i += len(u)
						continue
					}
				}
			}
			b.WriteByte(c)
			i++
		case ' ':
			// Two or more trailing spaces before a newline: hard break.
			j := i
			for j < n && s[j] == ' ' {
				j++
			}
			if j < n && s[j] == '\n' && j-i >= 2 {
				b.WriteString("<br>")
			} else if j < n && s[j] == '\n' {
				// plain soft break
			} else {
				b.WriteString(s[i:j])
			}
			i = j
		case '&':
			b.WriteString("&amp;")
			i++
		case '>':
			b.WriteString("&gt;")
			i++
		case '"':
			b.WriteString("&quot;")
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// findRun finds the next backtick run of exactly the same length as run.
func findRun(s string, from int, run string) int {
	for i := from; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(s) && s[j] == '`' {
			j++
		}
		if j-i == len(run) {
			return i
		}
		i = j
	}
	return -1
}

// link parses [text](url "title") and ![alt](url) at s[i].
func (r *mdRenderer) link(s string, i int) (string, int, bool) {
	image := s[i] == '!'
	open := i
	if image {
		if i+1 >= len(s) || s[i+1] != '[' {
			return "", 0, false
		}
		open = i + 1
	}
	// Find the matching ']' honoring nesting, code spans, and escapes.
	depth := 0
	closeIdx := -1
	for j := open; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '`':
			k := j
			for k < len(s) && s[k] == '`' {
				k++
			}
			if e := findRun(s, k, s[j:k]); e >= 0 {
				j = e + (k - j) - 1
			} else {
				j = k - 1
			}
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				closeIdx = j
			}
		}
		if closeIdx >= 0 {
			break
		}
	}
	if closeIdx < 0 || closeIdx+1 >= len(s) || s[closeIdx+1] != '(' {
		return "", 0, false
	}
	text := s[open+1 : closeIdx]

	// Parse destination up to the matching ')'.
	p := closeIdx + 2
	for p < len(s) && isSpaceByte(s[p]) {
		p++
	}
	var dest string
	if p < len(s) && s[p] == '<' {
		e := strings.IndexByte(s[p:], '>')
		if e < 0 {
			return "", 0, false
		}
		dest = s[p+1 : p+e]
		p += e + 1
	} else {
		start, parens := p, 0
		for p < len(s) && !isSpaceByte(s[p]) {
			if s[p] == '\\' && p+1 < len(s) {
				p += 2
				continue
			}
			if s[p] == '(' {
				parens++
			} else if s[p] == ')' {
				if parens == 0 {
					break
				}
				parens--
			}
			p++
		}
		dest = s[start:p]
	}
	for p < len(s) && isSpaceByte(s[p]) {
		p++
	}
	title := ""
	if p < len(s) && (s[p] == '"' || s[p] == '\'') {
		q := s[p]
		e := strings.IndexByte(s[p+1:], q)
		if e < 0 {
			return "", 0, false
		}
		title = s[p+1 : p+1+e]
		p += e + 2
		for p < len(s) && isSpaceByte(s[p]) {
			p++
		}
	}
	if p >= len(s) || s[p] != ')' {
		return "", 0, false
	}
	adv := p + 1 - i

	href := r.resolve(dest, image)
	titleAttr := ""
	if title != "" {
		titleAttr = ` title="` + html.EscapeString(title) + `"`
	}
	if image {
		if href == "" {
			return html.EscapeString(text), adv, true
		}
		return `<img src="` + html.EscapeString(href) + `" alt="` + html.EscapeString(stripInline(text)) + `"` + titleAttr + ">", adv, true
	}
	inner := r.inline(text)
	if href == "" {
		return inner, adv, true
	}
	return `<a href="` + html.EscapeString(href) + `"` + titleAttr + ">" + inner + "</a>", adv, true
}

// angle handles <autolinks>, allowlisted HTML tags, and comments at s[i].
func (r *mdRenderer) angle(s string, i int) (string, int, bool) {
	rest := s[i:]
	if strings.HasPrefix(rest, "<!--") {
		if e := strings.Index(rest, "-->"); e >= 0 {
			return "", e + 3, true
		}
		return "", 0, false
	}
	if m := reAutolink.FindStringSubmatch(rest); m != nil {
		if href := safeURL(m[1]); href != "" {
			return `<a href="` + html.EscapeString(href) + `">` + html.EscapeString(m[1]) + "</a>", len(m[0]), true
		}
	}
	m := reTag.FindStringSubmatch(rest)
	if m == nil {
		return "", 0, false
	}
	closing, name, attrs := m[1] == "/", strings.ToLower(m[2]), m[3]
	switch name {
	case "br", "hr":
		return "<" + name + ">", len(m[0]), true
	case "b", "i", "em", "strong", "code", "kbd", "sub", "sup", "del", "ins", "s", "u", "small", "mark",
		"p", "div", "details", "summary", "center", "h1", "h2", "h3", "h4", "h5", "h6",
		"table", "thead", "tbody", "tr", "ul", "ol", "li", "blockquote", "pre", "picture":
		if closing {
			return "</" + name + ">", len(m[0]), true
		}
		out := "<" + name
		if a := attrValues(attrs); a["align"] != "" && (name == "p" || name == "div" || strings.HasPrefix(name, "h")) {
			switch a["align"] {
			case "center", "left", "right":
				out += ` style="text-align:` + a["align"] + `"`
			}
		}
		return out + ">", len(m[0]), true
	case "td", "th":
		if closing {
			return "</" + name + ">", len(m[0]), true
		}
		return "<" + name + ">", len(m[0]), true
	case "a":
		if closing {
			return "</a>", len(m[0]), true
		}
		if href := r.resolve(attrValues(attrs)["href"], false); href != "" {
			return `<a href="` + html.EscapeString(href) + `">`, len(m[0]), true
		}
		return "<a>", len(m[0]), true
	case "img", "source":
		if closing {
			return "", len(m[0]), true
		}
		a := attrValues(attrs)
		src := a["src"]
		if name == "source" {
			return "", len(m[0]), true // not rendered; <img> fallback inside <picture> is used
		}
		href := r.resolve(src, true)
		if href == "" {
			return "", len(m[0]), true
		}
		out := `<img src="` + html.EscapeString(href) + `" alt="` + html.EscapeString(a["alt"]) + `"`
		for _, k := range []string{"width", "height"} {
			if v := a[k]; v != "" && reDim.MatchString(v) {
				out += " " + k + `="` + v + `"`
			}
		}
		return out + ">", len(m[0]), true
	}
	return "", 0, false
}

func attrValues(s string) map[string]string {
	out := map[string]string{}
	for _, m := range reAttr.FindAllStringSubmatch(s, -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		out[strings.ToLower(m[1])] = html.UnescapeString(v)
	}
	return out
}

// emphasis handles **strong**, *em*, __strong__, _em_, and ~~strike~~.
func (r *mdRenderer) emphasis(s string, i int) (string, int, bool) {
	c := s[i]
	j := i
	for j < len(s) && s[j] == c {
		j++
	}
	run := j - i
	if c == '~' {
		if run != 2 {
			return "", 0, false
		}
	} else if run > 3 {
		return "", 0, false
	}
	if j >= len(s) || isSpaceByte(s[j]) {
		return "", 0, false
	}
	if c == '_' && i > 0 && isWordByte(s[i-1]) {
		return "", 0, false
	}
	delim := s[i:j]
	for k := j; k < len(s); k++ {
		if s[k] == '`' { // skip code spans
			e := k
			for e < len(s) && s[e] == '`' {
				e++
			}
			if x := findRun(s, e, s[k:e]); x >= 0 {
				k = x + (e - k) - 1
			} else {
				k = e - 1
			}
			continue
		}
		if s[k] == '\\' {
			k++
			continue
		}
		if s[k] != c {
			continue
		}
		e := k
		for e < len(s) && s[e] == c {
			e++
		}
		if e-k >= len(delim) && !isSpaceByte(s[k-1]) && k > j {
			if c == '_' && e < len(s) && isWordByte(s[e]) {
				k = e - 1
				continue
			}
			// Use the last len(delim) characters of the run as the closer.
			closeAt := e - len(delim)
			inner := r.inline(s[j:closeAt])
			switch {
			case c == '~':
				return "<del>" + inner + "</del>", e - i, true
			case len(delim) == 1:
				return "<em>" + inner + "</em>", e - i, true
			case len(delim) == 2:
				return "<strong>" + inner + "</strong>", e - i, true
			default:
				return "<em><strong>" + inner + "</strong></em>", e - i, true
			}
		}
		k = e - 1
	}
	return "", 0, false
}

// safeURL allows http(s), mailto, anchors, and relative URLs only.
func safeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if i := strings.IndexAny(u, ":/?#"); i >= 0 && u[i] == ':' {
		switch strings.ToLower(u[:i]) {
		case "http", "https", "mailto":
			return u
		}
		return ""
	}
	return u
}
