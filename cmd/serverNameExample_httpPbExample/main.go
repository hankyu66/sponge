// Package main is the http server of the application.
package main

import (
	"github.com/hankyu66/sponge/cmd/serverNameExample_httpPbExample/initial"

	"github.com/hankyu66/sponge/pkg/app"
)

func main() {
	initial.Config()
	servers := initial.RegisterServers()
	closes := initial.RegisterClose(servers)

	a := app.New(servers, closes)
	a.Run()
}
