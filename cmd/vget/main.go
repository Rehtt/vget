package main

import (
	"os"

	"github.com/guiyumin/vget/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
