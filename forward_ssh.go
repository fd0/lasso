package main

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

// SSHConnection describes one SSH connection to a server.
type SSHConnection struct {
	Server       string `hcl:"server"`
	RemoteListen string `hcl:"remote_listen"`
	User         string `hcl:"user"`
	Hostkey      string `hcl:"hostkey"`
	Key          string `hcl:"key"`
}

func (c SSHConnection) connect(target string) (success bool, err error) {
	signer, err := ssh.ParsePrivateKey([]byte(c.Key))
	if err != nil {
		printErr("[ssh %v] unable to parse private key: %v\n", c.Server, err)
		return false, err
	}

	hostkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(c.Hostkey))
	if err != nil {
		printErr("[ssh %v] unable to parse host key: %v\n", c.Server, err)
		return false, err
	}

	clientCfg := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(hostkey),
		HostKeyAlgorithms: []string{
			hostkey.Type(),
		},
		Timeout: opts.ConnectTimeout,
	}

	client, err := ssh.Dial("tcp", c.Server, clientCfg)
	if err != nil {
		printErr("[ssh %v] error: %v\n", c.Server, err)
		return false, err
	}

	verbose("[ssh %v] connected\n", c.Server)

	remoteListen, err := client.Listen("tcp", c.RemoteListen)
	if err != nil {
		printErr("[ssh %v] unable to listen remotely on %v: %v\n", c.Server, c.RemoteListen, err)
		_ = client.Close()
		return true, err
	}

	wg := &errgroup.Group{}
	for {
		incoming, err := remoteListen.Accept()
		if err != nil {
			printErr("[ssh %v] error accepting incoming remote connection: %v\n", c.Server, err)
			break
		}

		verbose("[ssh %v] new incoming connection from remote %v, err %v\n", c.Server, incoming.RemoteAddr(), err)

		conn, err := net.Dial("tcp", target)
		if err != nil {
			printErr("[ssh %v] unable to connect to target %v: %v\n", c.Server, target, err)
			_ = client.Close()
			return true, err
		}

		verbose("[ssh %v] success, connected to target %v, start forwarding data\n", c.Server, target)
		forward(wg, incoming, conn)
	}

	err = wg.Wait()
	if err != nil {
		_ = client.Close()
		return true, err
	}

	return true, client.Close()
}

// Forward connects to an SSH server and creates a new remote port forward to target.
func (c SSHConnection) Forward(wg *errgroup.Group, target string) {
	wg.Go(func() error {
		for {
			success, err := c.connect(target)
			if err != nil {
				printErr("[ssh %v] error: %v\n", c.Server, err)
			}

			if success {
				time.Sleep(opts.ReconnectDelay)
			} else {
				time.Sleep(opts.BackoffDelay)
			}
		}
	})
}
