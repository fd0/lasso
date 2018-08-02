package main

import (
	"io"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

func forward(wg *errgroup.Group, c1, c2 io.ReadWriteCloser) {
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
}

// TCPConnection describes one plain tcp connection to a server.
type TCPConnection struct {
	Server string `hcl:"server"`
}

// connect connects the two endpoints and forwards data between them. If the
// connection to the outbound endpoint could be established successfully at
// some point, success is set to true.
func (c TCPConnection) connect(target string) (success bool, err error) {

	c1, err := net.DialTimeout("tcp", c.Server, opts.ConnectTimeout)
	if err != nil {
		return false, err
	}

	verbose("[tcp %v] success, connected\n", c.Server)
	verbose("[tcp %v] connecting target %v\n", c.Server, target)

	c2, err := net.Dial("tcp", target)
	if err != nil {
		_ = c1.Close()
		return true, err
	}

	verbose("[tcp %v] success, connected to target %v, start forwarding data\n", c.Server, target)
	wg := &errgroup.Group{}
	forward(wg, c1, c2)
	return true, wg.Wait()
}

// Forward connects the two endpoints, it creates multiple Goroutines with the
// given wg.
func (c TCPConnection) Forward(wg *errgroup.Group, target string) {
	wg.Go(func() error {
		for {
			success, err := c.connect(target)
			if err != nil {
				printErr("[tcp %v] connection died, error: %v, sleeping\n", c.Server, err)
			}

			delay := opts.BackoffDelay
			if success {
				delay = opts.ReconnectDelay
			}

			time.Sleep(delay)
		}
	})
}
