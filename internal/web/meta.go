package web

import (
	"encoding/json"
	"log"
	"net/url"
	"regexp"
	"strings"
)

// MetaFile is the optional project metadata file at the root of a repository.
const MetaFile = ".homegit-meta"

const (
	maxMetaBytes       = 16 << 10
	maxDescriptionRune = 350
	maxHomepageLen     = 2048
	maxTopics          = 20
)

var reTopic = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`)

// Meta is display-only project metadata read from MetaFile. It never affects
// access or behavior, and all values are treated as untrusted text.
type Meta struct {
	Description string
	Homepage    string
	Topics      []string
}

// IsEmpty reports whether m carries nothing to display.
func (m *Meta) IsEmpty() bool {
	return m == nil || (m.Description == "" && m.Homepage == "" && len(m.Topics) == 0)
}

// HomepageHost returns the host part of the homepage for display.
func (m *Meta) HomepageHost() string {
	u, err := url.Parse(m.Homepage)
	if err != nil {
		return m.Homepage
	}
	return u.Host + strings.TrimRight(u.Path, "/")
}

// ParseMeta parses and sanitizes a MetaFile. Unknown keys are ignored so the
// format can grow. Invalid values are dropped individually: a bad homepage
// does not discard the description.
func ParseMeta(data []byte) (*Meta, error) {
	if len(data) > maxMetaBytes {
		return nil, errMetaTooLarge
	}
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))
	var raw struct {
		Description string   `json:"description"`
		Homepage    string   `json:"homepage"`
		Topics      []string `json:"topics"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	m := &Meta{Description: strings.Join(strings.Fields(raw.Description), " ")}
	if r := []rune(m.Description); len(r) > maxDescriptionRune {
		m.Description = string(r[:maxDescriptionRune])
	}

	if hp := strings.TrimSpace(raw.Homepage); hp != "" && len(hp) <= maxHomepageLen {
		if u, err := url.Parse(hp); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			m.Homepage = hp
		}
	}

	seen := map[string]bool{}
	for _, t := range raw.Topics {
		t = strings.ToLower(strings.TrimSpace(t))
		if !reTopic.MatchString(t) || seen[t] {
			continue
		}
		seen[t] = true
		m.Topics = append(m.Topics, t)
		if len(m.Topics) == maxTopics {
			break
		}
	}
	return m, nil
}

type metaError string

func (e metaError) Error() string { return string(e) }

const errMetaTooLarge = metaError("metadata file is too large")

// ReadMeta returns the repository's metadata, or nil when the repo has no
// MetaFile or it is invalid (invalid files are logged and ignored).
func (s *Store) ReadMeta(name string) *Meta {
	data, err := s.Blob(name, MetaFile, maxMetaBytes+1)
	if err != nil {
		return nil // missing file, empty repo, or unreadable: nothing to show
	}
	m, err := ParseMeta(data)
	if err != nil {
		log.Printf("web: %s: ignoring %s: %v", name, MetaFile, err)
		return nil
	}
	if m.IsEmpty() {
		return nil
	}
	return m
}
