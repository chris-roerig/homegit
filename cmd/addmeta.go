package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chris-roerig/homegit/internal/web"
)

// AddMeta creates a .homegit-meta file in the root of the git repository
// containing dir. Values come from flags; when none are given it prompts for
// them on in/out. Values are sanitized with the same rules the web UI uses, and
// anything dropped is reported.
func AddMeta(args []string, dir string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("add-meta", flag.ContinueOnError)
	fs.SetOutput(out)
	description := fs.String("description", "", "short project description")
	homepage := fs.String("homepage", "", "project homepage (http or https URL)")
	topics := fs.String("topics", "", "comma-separated topics, e.g. go,cli")
	force := fs.Bool("force", false, "overwrite an existing "+web.MetaFile)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q (use --description, --homepage, --topics)", fs.Arg(0))
	}

	root, err := gitRoot(dir)
	if err != nil {
		return err
	}
	target := filepath.Join(root, web.MetaFile)
	if _, err := os.Stat(target); err == nil && !*force {
		return fmt.Errorf("%s already exists (edit it, or use --force to overwrite)", target)
	}

	flagsGiven := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name != "force" {
			flagsGiven = true
		}
	})
	if !flagsGiven {
		r := bufio.NewReader(in)
		fmt.Fprintln(out, "Create "+web.MetaFile+" (press Enter to skip a field)")
		*description = prompt(r, out, "Description: ")
		*homepage = prompt(r, out, "Homepage URL: ")
		*topics = prompt(r, out, "Topics (comma-separated, e.g. go,cli): ")
	}

	raw, err := json.Marshal(map[string]any{
		"description": *description,
		"homepage":    *homepage,
		"topics":      splitTopics(*topics),
	})
	if err != nil {
		return err
	}
	meta, err := web.ParseMeta(raw)
	if err != nil {
		return err
	}
	for _, w := range metaWarnings(*description, *homepage, *topics, meta) {
		fmt.Fprintln(out, "Warning:", w)
	}
	if meta.IsEmpty() {
		return errors.New("nothing to write: provide at least a description, homepage, or topic")
	}

	data, err := meta.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n✓ Wrote %s\n\n", target)
	fmt.Fprintln(out, "Commit and push it so the web UI can show it:")
	fmt.Fprintf(out, "  git add %s && git commit -m \"Add project metadata\" && git push\n", web.MetaFile)
	return nil
}

func gitRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", errors.New("not inside a git repository (run 'homegit init' first)")
	}
	return strings.TrimSpace(string(out)), nil
}

func prompt(r *bufio.Reader, out io.Writer, label string) string {
	fmt.Fprint(out, label)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func splitTopics(s string) []string {
	var topics []string
	for _, t := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
		topics = append(topics, t)
	}
	return topics
}

// metaWarnings describes input that sanitizing removed or changed.
func metaWarnings(description, homepage, topics string, m *web.Meta) []string {
	var w []string
	if strings.TrimSpace(homepage) != "" && m.Homepage == "" {
		w = append(w, fmt.Sprintf("ignored homepage %q (must be an http or https URL)", strings.TrimSpace(homepage)))
	}
	kept := map[string]bool{}
	for _, t := range m.Topics {
		kept[t] = true
	}
	var dropped []string
	for _, t := range splitTopics(topics) {
		if !kept[strings.ToLower(t)] {
			dropped = append(dropped, t)
		}
	}
	if len(dropped) > 0 {
		w = append(w, fmt.Sprintf("ignored topics %s (use up to 20 lowercase letters, digits, and hyphens)", strings.Join(dropped, ", ")))
	}
	if n := len([]rune(strings.Join(strings.Fields(description), " "))); n > len([]rune(m.Description)) {
		w = append(w, fmt.Sprintf("description shortened to %d characters", len([]rune(m.Description))))
	}
	return w
}
