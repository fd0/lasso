package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

// Options collects global configuration.
type Options struct {
	Quiet          bool
	TCPServer      []string
	Target         string
	ReconnectDelay time.Duration
	BackoffDelay   time.Duration
}

var (
	opts  Options
	flags *pflag.FlagSet
)

func print(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

func verbose(msg string, args ...interface{}) {
	if opts.Quiet {
		return
	}

	fmt.Printf(msg, args...)
}

func printErr(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
}

func main() {
	flags = pflag.NewFlagSet("connectbackd", pflag.ContinueOnError)
	flags.BoolVar(&opts.Quiet, "quiet", false, "Be quiet, only print errors")
	flags.StringArrayVar(&opts.TCPServer, "tcp", nil, "Connect back to `host:port` via plain TCP (can be specified multiple times)")
	flags.StringVar(&opts.Target, "target", "localhost:22", "Connect to `host:port`")
	flags.DurationVar(&opts.ReconnectDelay, "reconnect", 2*time.Second, "Wait for `duration` before reconnecting")
	flags.DurationVar(&opts.BackoffDelay, "backoff", 10*time.Second, "Wait for `duration` before trying to connect")

	err := flags.Parse(os.Args)
	if err == pflag.ErrHelp {
		os.Exit(0)
	}

	if err != nil {
		printErr("error parsing flags: %v\n", err)
		os.Exit(1)
	}

	for _, srv := range opts.TCPServer {
		if !strings.Contains(srv, ":") {
			printErr("Arong format for --server, need host:port\n")
			os.Exit(1)
		}
	}

	// connect back to plain tcp ports
	wg := &errgroup.Group{}
	for _, server := range opts.TCPServer {
		forwardPlainTCP(wg, server, opts.Target)
	}

	err = wg.Wait()
	if err != nil {
		printErr("error: %v\n", err)
		os.Exit(1)
	}
}
