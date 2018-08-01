package main

import (
	"fmt"
	"io"
	"net"
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

// connectPlainTCP connects the two endpoints and forwards data between them.
// If the connection to the outbound endpoint could be established successfully
// at some point, success is set to true.
func connectPlainTCP(outbound, target string) (success bool, err error) {
	verbose("[%v] connecting\n", outbound)

	c1, err := net.Dial("tcp", outbound)
	if err != nil {
		return false, err
	}

	verbose("[%v] success, connected\n", outbound)
	verbose("[%v] connecting target %v\n", outbound, target)

	c2, err := net.Dial("tcp", target)
	if err != nil {
		_ = c1.Close()
		return true, err
	}

	verbose("[%v] success, connected to target %v, start forwarding data\n", outbound, target)

	wg := &errgroup.Group{}

	wg.Go(func() error {
		_, err := io.Copy(c2, c1)
		if err != nil {
			_ = c2.Close()
			return err
		}
		return c2.Close()
	})

	wg.Go(func() error {
		_, err := io.Copy(c1, c2)
		if err != nil {
			_ = c1.Close()
			return err
		}
		return c1.Close()
	})

	return true, wg.Wait()
}

func forwardPlainTCP(wg *errgroup.Group, outbound, target string) {
	wg.Go(func() error {
		for {
			success, err := connectPlainTCP(outbound, target)
			if err != nil {
				printErr("[%v] connection died, error: %v, sleeping\n", outbound, err)
			}

			if success {
				verbose("reconnecting after %v\n", opts.ReconnectDelay)
				time.Sleep(opts.ReconnectDelay)
			} else {
				verbose("reconnecting after %v\n", opts.BackoffDelay)
				time.Sleep(opts.BackoffDelay)
			}
		}
	})
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
