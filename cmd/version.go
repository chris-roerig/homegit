package cmd

import (
	"fmt"
)

const Version = "1.0.3"

func ShowVersion() {
	fmt.Printf("homegit version %s\n", Version)
}
