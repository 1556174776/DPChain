package main

import (
	"os"
	"p2p-go/cli"
)

func main() {
	defer os.Exit(0)
	cmd := cli.NewCommandLine()
	cmd.Run()
}
