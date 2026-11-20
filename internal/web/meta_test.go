package web

import (
	"strings"
	"testing"
)

func TestParseMeta(t *testing.T) {
	m, err := ParseMeta([]byte(`{
		"description": "  A   tiny\n tool  ",
		"homepage": "https://example.com/docs/",
		"topics": ["Go", "git-server", "go", "bad topic", "-x", "", "ok2"],
		"future_key": {"nested": true}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Description != "A tiny tool" {
		t.Errorf("description = %q", m.Description)
	}
	if m.Homepage != "https://example.com/docs/" || m.HomepageHost() != "example.com/docs" {
		t.Errorf("homepage = %q host = %q", m.Homepage, m.HomepageHost())
	}
	if got := strings.Join(m.Topics, ","); got != "go,git-server,ok2" {
		t.Errorf("topics = %q", got)
	}
}

func TestParseMetaDropsBadValuesIndividually(t *testing.T) {
	for _, hp := range []string{"javascript:alert(1)", "ftp://x.com", "//x.com", "example.com", "http://"} {
		m, err := ParseMeta([]byte(`{"description":"keep","homepage":"` + hp + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if m.Homepage != "" || m.Description != "keep" {
			t.Errorf("homepage %q: got %+v", hp, m)
		}
	}
}

func TestParseMetaLimits(t *testing.T) {
	m, _ := ParseMeta([]byte(`{"description":"` + strings.Repeat("é", 1000) + `"}`))
	if n := len([]rune(m.Description)); n != maxDescriptionRune {
		t.Errorf("description runes = %d", n)
	}
	var topics []string
	for i := 0; i < 50; i++ {
		topics = append(topics, `"t`+strings.Repeat("a", i)+`"`)
	}
	m, _ = ParseMeta([]byte(`{"topics":[` + strings.Join(topics, ",") + `]}`))
	if len(m.Topics) != maxTopics {
		t.Errorf("topics = %d", len(m.Topics))
	}
	if _, err := ParseMeta([]byte(strings.Repeat(" ", maxMetaBytes+1))); err == nil {
		t.Error("oversized file should error")
	}
	for _, bad := range []string{"", "not json", "[1,2]", `{"description": 5}`} {
		if _, err := ParseMeta([]byte(bad)); err == nil {
			t.Errorf("ParseMeta(%q) should error", bad)
		}
	}
}

func TestMetaIsEmpty(t *testing.T) {
	var nilMeta *Meta
	if !nilMeta.IsEmpty() || !(&Meta{}).IsEmpty() || (&Meta{Topics: []string{"a"}}).IsEmpty() {
		t.Error("IsEmpty misbehaves")
	}
}

func TestAboutAndHomeDescriptions(t *testing.T) {
	s := newTestServer(t)

	root := get(s, "/demo/tree/").Body.String()
	for _, want := range []string{
		`<h2>About</h2>`,
		`A &lt;b&gt;demo&lt;/b&gt; project`,
		`<a href="https://example.com/demo" rel="noopener noreferrer nofollow">example.com/demo</a>`,
		`<span class="topic">go</span>`,
		`git clone ssh://box.local:2222/demo.git`,
	} {
		if !strings.Contains(root, want) {
			t.Errorf("root page missing %q", want)
		}
	}
	if strings.Contains(root, "<b>demo</b>") {
		t.Error("description must be escaped")
	}

	if sub := get(s, "/demo/tree/src").Body.String(); strings.Contains(sub, "<h2>About</h2>") {
		t.Error("About must only appear on the repo root")
	}

	home := get(s, "/").Body.String()
	if !strings.Contains(home, `<p class="desc">A &lt;b&gt;demo&lt;/b&gt; project</p>`) {
		t.Errorf("home page missing description:\n%s", home)
	}

	// A repo without metadata shows a placeholder, not an error.
	if blank := get(s, "/blank/tree/"); blank.Code != 200 {
		t.Errorf("blank status %d", blank.Code)
	}
	if s.store.ReadMeta("blank") != nil || s.store.ReadMeta("nope") != nil {
		t.Error("ReadMeta should be nil for empty/missing repos")
	}
}
