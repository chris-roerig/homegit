package web

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string // substrings that must appear
		not  []string // substrings that must not appear
	}{
		{"heading", "# Hello *World*", []string{`<h1 id="hello-world">Hello <em>World</em></h1>`}, nil},
		{"setext", "Title\n=====\n\ntext", []string{`<h1 id="title">Title</h1>`, "<p>text</p>"}, nil},
		{"paragraph joins lines", "a\nb", []string{"<p>a\nb</p>"}, nil},
		{"strong em code", "**b** _i_ `c<d>`", []string{"<strong>b</strong>", "<em>i</em>", "<code>c&lt;d&gt;</code>"}, nil},
		{"snake_case untouched", "my_var_name", []string{"my_var_name"}, []string{"<em>"}},
		{"strike", "~~x~~", []string{"<del>x</del>"}, nil},
		{"link", `[go](https://go.dev "Go")`, []string{`<a href="https://go.dev" title="Go">go</a>`}, nil},
		{"image link badge", "[![b](https://x/y.svg)](https://x)", []string{`<a href="https://x"><img src="https://x/y.svg" alt="b"></a>`}, nil},
		{"bare url", "see https://example.com/a.", []string{`<a href="https://example.com/a">https://example.com/a</a>.`}, nil},
		{"javascript link dropped", "[x](javascript:alert(1))", []string{">x"}, []string{"javascript", "<a"}},
		{"raw script escaped", "<script>alert(1)</script>", []string{"&lt;script&gt;"}, []string{"<script>"}},
		{"onerror stripped", `<img src="a.png" onerror="x()">`, []string{`<img src="a.png"`}, []string{"onerror"}},
		{"fence", "```go\nx := 1 < 2\n```", []string{`<pre><code class="language-go">x := 1 &lt; 2`}, nil},
		{"fence keeps markdown", "```\n# not heading\n```", []string{"# not heading"}, []string{"<h1"}},
		{"ul", "- a\n- b", []string{"<ul>", "<li>a</li>", "<li>b</li>"}, []string{"<p>"}},
		{"ol start", "3. a\n4. b", []string{`<ol start="3">`}, nil},
		{"nested list", "- a\n  - b\n  - c\n- d", []string{"<li>a\n<ul>", "<li>b</li>", "<li>d</li>"}, nil},
		{"loose list", "- a\n\n- b", []string{"<p>a</p>", "<p>b</p>"}, nil},
		{"task", "- [x] done\n- [ ] todo", []string{`class="task-list"`, `disabled checked> done`, `disabled> todo`}, nil},
		{"blockquote", "> a\n> b", []string{"<blockquote>", "<p>a\nb</p>"}, nil},
		{"hr", "a\n\n---\n\nb", []string{"<hr>"}, nil},
		{"table", "| a | b |\n|:--|--:|\n| 1 | 2 |", []string{"<th style=\"text-align:left\">a</th>", "<td style=\"text-align:right\">2</td>"}, nil},
		{"indented code", "    code <x>", []string{"<pre><code>code &lt;x&gt;"}, nil},
		{"hard break", "a  \nb", []string{"a<br>\nb"}, nil},
		{"comment dropped", "<!-- hidden -->\n\ntext", []string{"<p>text</p>"}, []string{"hidden"}},
		{"html block", "<p align=\"center\">\n<img src=\"l.png\" width=\"100\">\n</p>", []string{`<p style="text-align:center">`, `<img src="l.png" alt="" width="100">`}, nil},
		{"escape", `\*lit\*`, []string{"*lit*"}, []string{"<em>"}},
		{"heading ids dedupe", "# A\n\n# A", []string{`id="a"`, `id="a-1"`}, nil},
		{"unclosed fence", "```\nabc", []string{"<pre><code>abc"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdown(tt.in, nil)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q in:\n%s", w, got)
				}
			}
			for _, n := range tt.not {
				if strings.Contains(got, n) {
					t.Errorf("unexpected %q in:\n%s", n, got)
				}
			}
		})
	}
}

func TestRenderMarkdownNeverPanics(t *testing.T) {
	inputs := []string{"", "[", "![", "[a](", "[a](b", "**", "*a", "`", "```", "> ", "- ", "1.", "|", "| a |\n|-|", "<", "<a", "<a href=", "\\", "![a](b \"c", "- a\n\n  b\n\n- c", "* * *", "[a]( <b )"}
	for _, in := range inputs {
		RenderMarkdown(in, nil)
	}
}
