package main

import (
	"fmt"
	"os"

	"s88/wp/backend"
	"s88/wp/backend/server"

	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	HelpName: "webprops",
	Usage:    "Web Properties API Server",
	Version:  server.Tag,
	Commands: []*cli.Command{
		backend.Command,
	},
}

func main() {
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
