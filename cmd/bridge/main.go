package main

import (
	"os"

	"github.com/felixgeelhaar/bridge/internal/interfaces/cli"
)

func main() {
	app := cli.NewApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
