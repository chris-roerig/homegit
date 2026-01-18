package cmd

import (
	"fmt"
)

const Version = "1.0.0"

func ShowVersion() {
	fmt.Printf("homegit version %s\n", Version)
}
