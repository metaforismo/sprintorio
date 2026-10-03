package main

import (
	"context"
	"github.com/metaforismo/sprintorio/BE/internal/agentclient"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(agentclient.RunCLI(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
