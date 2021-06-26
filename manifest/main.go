package main

import (
	"context"
	"flag"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
)

func main() {
	var debug = flag.Bool("debug", false, "run the provider in re-attach mode")
	var jsonLog = flag.Bool("jsonlog", false, "output logging as JSON")

	flag.Parse()
	ctx := context.Background()

	var logLevel string
	logLevel, ok := os.LookupEnv("TF_LOG")
	if !ok {
		logLevel = "info"
	}

	logger := hclog.New(&hclog.LoggerOptions{
		JSONFormat: *jsonLog,
		Level:      hclog.LevelFromString(logLevel),
		Output:     os.Stderr,
	})

	if *debug {
		provider.ServeReattach(ctx, logger)
	} else {
		provider.Serve(ctx, logger)
	}
}
