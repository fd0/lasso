package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

// Options collects global configuration.
type Options struct {
	Quiet      bool
	ConfigFile string

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
	flags.StringVar(&opts.ConfigFile, "config", "", "Load configuration from `filename`")

	err := flags.Parse(os.Args)
	if err == pflag.ErrHelp {
		os.Exit(0)
	}

	if err != nil {
		printErr("error parsing flags: %v\n", err)
		os.Exit(1)
	}

	var cfg Config
	if opts.ConfigFile != "" {
		verbose("loading config %v\n", opts.ConfigFile)
		cfg, err = ParseConfig(opts.ConfigFile)
		if err != nil {
			printErr("error parsing config file: %v\n", err)
			os.Exit(1)
		}
	}

	opts.BackoffDelay = time.Duration(cfg.BackoffDelay) * time.Second
	opts.ReconnectDelay = time.Duration(cfg.ReconnectDelay) * time.Second

	wg := &errgroup.Group{}

	// connect to plain tcp ports
	for _, tcp := range cfg.TCP {
		forwardPlainTCP(wg, tcp.Server, cfg.Target)
	}

	// connect to SSH servers
	for _, ssh := range cfg.SSH {
		forwardSSH(ssh, cfg.Target)
	}

	err = wg.Wait()
	if err != nil {
		printErr("error: %v\n", err)
		os.Exit(1)
	}
}
