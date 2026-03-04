package main

import (
	"os"

	"github.com/withObsrvr/prism/cmd/prism"
)

func main() {
	if err := prism.Execute(); err != nil {
		os.Exit(1)
	}
}
