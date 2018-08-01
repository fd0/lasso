package main

import (
	"io"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

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

// forwardPlainTCP connects the two endpoints, it creates multiple Goroutines
// with the given Group.
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
