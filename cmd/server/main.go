package main

import (
	"os"

	"github.com/sunary/aku/config"
	"github.com/sunary/aku/gateway"
	"github.com/sunary/aku/loging"
)

var (
	ll = loging.New()
)

func run(_ []string) error {
	cfg := config.Load()
	gw := gateway.NewGateway(cfg)
	return gw.Start()
}

func main() {
	if err := run(os.Args); err != nil {
		ll.Fatal("aku start", loging.Err(err))
	}
}
