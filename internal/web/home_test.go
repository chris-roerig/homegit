package web

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHomeListMarkup(t *testing.T) {
	s := newTestServer(t)
	rec := get(s, "/")
	body := rec.Body.String()

	for _, want := range []string{
		`<li data-repo data-name="demo" data-text="a &lt;b&gt;demo&lt;/b&gt; project go git">`,
		`<li data-repo data-name="blank" data-text="">`,
		`<span class="topic">go</span>`,
		`<code class="cloneurl">ssh://box.local:2222/demo.git</code>`,
		`<input type="search" id="repo-filter"`,
		`<script src="/_static/filter.js" defer></script>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("home page missing %q\n%s", want, body)
		}
	}
	// The filter starts hidden so it only appears when scripting works.
	if !strings.Contains(body, `id="repo-filter-wrap" hidden`) {
		t.Error("filter should be hidden until the script enables it")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") || strings.Contains(csp, "unsafe") {
		t.Errorf("CSP should allow same-origin scripts only: %q", csp)
	}
	if strings.Contains(body, "<script>") {
		t.Error("no inline scripts allowed")
	}

	js := get(s, "/_static/filter.js")
	if js.Code != 200 || !strings.Contains(js.Body.String(), "repo-filter") {
		t.Errorf("filter.js not served: %d", js.Code)
	}
}

func TestHomeEmptyHasNoFilter(t *testing.T) {
	s, err := New(t.TempDir(), 2222, "")
	if err != nil {
		t.Fatal(err)
	}
	body := get(s, "/").Body.String()
	if strings.Contains(body, "repo-filter") || !strings.Contains(body, "No repositories yet") {
		t.Errorf("empty home page wrong:\n%s", body)
	}
}

// Runs the real filter.js against a minimal fake DOM. Skipped without Node.
func TestFilterScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	script := filepath.Join(t.TempDir(), "run.js")
	if err := os.WriteFile(script, []byte(filterHarness), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, script, filepath.Join("static", "filter.js"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("filter.js behavior failed: %v\n%s", err, out)
	}
}

const filterHarness = `
const fs = require('fs'), vm = require('vm'), assert = require('assert');
const src = fs.readFileSync(process.argv[2], 'utf8');

function item(name, text) {
  return { hidden: false, getAttribute: (k) => ({ 'data-name': name, 'data-text': text })[k] };
}
const items = [
  item('buenobot', 'an autonomous llm agent ollama qdrant'),
  item('Homegit', 'personal git server go'),
  item('port_man', ''),
];
const handlers = {};
const input = { value: '', addEventListener: (e, f) => { handlers[e] = f; } };
const els = {
  'repo-filter': input,
  'repo-filter-wrap': { hidden: true },
  'repo-list': { querySelectorAll: () => items },
  'repo-count': { textContent: '3' },
  'repo-empty': { hidden: true },
};
vm.runInNewContext(src, { document: { getElementById: (id) => els[id] } });

const visible = () => items.filter((i) => !i.hidden).map((i) => i.getAttribute('data-name'));
function type(v) { input.value = v; handlers.input(); }

assert.strictEqual(els['repo-filter-wrap'].hidden, false, 'filter should be revealed');
assert.deepStrictEqual(visible(), ['buenobot', 'Homegit', 'port_man']);

type('bbot');               assert.deepStrictEqual(visible(), ['buenobot']);        // fuzzy name
type('HGT');                assert.deepStrictEqual(visible(), ['Homegit']);         // case-insensitive
type('qdrant');             assert.deepStrictEqual(visible(), ['buenobot']);        // description/topic substring
type('git go');             assert.deepStrictEqual(visible(), ['Homegit']);         // every word must match
type('pm');                 assert.deepStrictEqual(visible(), ['port_man']);
assert.strictEqual(els['repo-count'].textContent, '1 of 3');
assert.strictEqual(els['repo-empty'].hidden, true);

type('zzz');                assert.deepStrictEqual(visible(), []);
assert.strictEqual(els['repo-empty'].hidden, false, 'empty message when nothing matches');

type('   ');                assert.strictEqual(visible().length, 3);                // whitespace = no filter
assert.strictEqual(els['repo-count'].textContent, '3');

type('bbot'); handlers.keydown({ key: 'Escape' });
assert.strictEqual(input.value, '');                assert.strictEqual(visible().length, 3);
console.log('ok');
`
