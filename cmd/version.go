package cmd

import (
	"fmt"
)

// Version is the homegit version. It defaults to the value below for local and
// development builds, but release builds override it via -ldflags
// "-X main.version=..." which is propagated here through SetVersion from
// package main. Keep this in sync with the latest released tag.
var Version = "1.0.5"

// SetVersion overrides the reported version. It is called from package main
// when a version string has been injected at build time via ldflags. An empty
// value is ignored so the compiled-in default is preserved for dev builds.
func SetVersion(v string) {
	if v != "" {
		Version = v
	}
}

func ShowVersion() {
	fmt.Printf("homegit version %s\n", Version)
}
