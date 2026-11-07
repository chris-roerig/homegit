package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	updateRepo       = "chris-roerig/homegit"
	updateHTTPClient = 30 * time.Second
)

// Update checks GitHub Releases for a newer version and, if found, replaces the
// running binary in place. It refuses to self-update when installed via
// Homebrew, directing the user to `brew upgrade` instead.
func Update() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	// Refuse to fight Homebrew: if this binary lives under a Homebrew prefix,
	// the correct update path is `brew upgrade`.
	if isHomebrewInstall(exe) {
		fmt.Println("homegit was installed via Homebrew.")
		fmt.Println("Update it with:")
		fmt.Println("  brew upgrade homegit")
		return nil
	}

	fmt.Printf("%s checking for updates...\n", versionLabel())

	latest, err := latestReleaseTag()
	if err != nil {
		return fmt.Errorf("could not check latest release: %w", err)
	}

	current := "v" + Version
	if !isNewer(latest, current) {
		fmt.Printf("Already up to date (%s).\n", current)
		return nil
	}

	fmt.Printf("Updating %s -> %s\n", current, latest)

	asset := fmt.Sprintf("homegit-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}
	base := fmt.Sprintf("https://github.com/%s/releases/download/%s", updateRepo, latest)

	// Download the new binary and checksums.
	binData, err := httpGetBytes(base + "/" + asset)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", asset, err)
	}
	if err := verifyDownloadedChecksum(binData, base+"/checksums.txt", asset); err != nil {
		return err
	}

	if err := replaceExecutable(exe, binData); err != nil {
		return err
	}

	fmt.Printf("%s updated to %s\n", versionLabel(), latest)
	fmt.Println("Restart any running server (e.g. 'homegit restart') to use the new version.")
	return nil
}

// isHomebrewInstall reports whether the executable path indicates a Homebrew
// installation. Homebrew installs live under its prefix (e.g.
// /opt/homebrew or /usr/local) inside a Cellar/opt layout.
func isHomebrewInstall(exe string) bool {
	if strings.Contains(exe, "/Cellar/") || strings.Contains(exe, "/homebrew/") {
		return true
	}
	// Linuxbrew / custom prefixes.
	if strings.Contains(exe, "/.linuxbrew/") {
		return true
	}
	return false
}

// latestReleaseTag queries the GitHub API for the latest release tag.
func latestReleaseTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", updateRepo)
	data, err := httpGetBytes(url)
	if err != nil {
		return "", err
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("could not parse release info: %w", err)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("no tag_name in latest release")
	}
	return payload.TagName, nil
}

// verifyDownloadedChecksum validates binData against the published checksums.
func verifyDownloadedChecksum(binData []byte, checksumsURL, asset string) error {
	data, err := httpGetBytes(checksumsURL)
	if err != nil {
		// Checksums are strongly preferred; treat a fetch failure as fatal for
		// a self-update since we're about to replace a running binary.
		return fmt.Errorf("could not download checksums for verification: %w", err)
	}
	var want string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, asset) {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				want = fields[0]
			}
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum listed for %s", asset)
	}
	sum := sha256.Sum256(binData)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("checksum mismatch for %s (expected %s, got %s)", asset, want, got)
	}
	return nil
}

// replaceExecutable atomically replaces the file at path with newData by
// writing to a temp file in the same directory and renaming over the original.
func replaceExecutable(path string, newData []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".homegit-update-*")
	if err != nil {
		return fmt.Errorf("could not create temp file in %s: %w (insufficient permissions?)", dir, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op if the rename succeeded

	if _, err := tmp.Write(newData); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write update: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not finalize update: %w", err)
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("could not set permissions on update: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("could not replace %s: %w (insufficient permissions?)", path, err)
	}
	return nil
}

// httpGetBytes performs a GET and returns the body, failing on non-2xx status.
func httpGetBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: updateHTTPClient}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "homegit-updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %s for %s", resp.Status, url)
	}
	return io.ReadAll(resp.Body)
}

// isNewer reports whether tag latest represents a higher semantic version than
// current. Both are expected in the form "vX.Y.Z".
func isNewer(latest, current string) bool {
	return compareVersions(latest, current) > 0
}

// compareVersions compares two "vX.Y.Z" version strings. Returns 1 if a > b,
// -1 if a < b, 0 if equal. Non-numeric or missing components are treated as 0.
func compareVersions(a, b string) int {
	as := parseVersion(a)
	bs := parseVersion(b)
	for i := 0; i < 3; i++ {
		if as[i] > bs[i] {
			return 1
		}
		if as[i] < bs[i] {
			return -1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// Drop any pre-release/build suffix.
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		n := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				n = 0
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}
