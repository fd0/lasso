package main

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

func connectSSH(cfg SSHConnection, target string) (success bool, err error) {
	signer, err := ssh.ParsePrivateKey([]byte(cfg.Key))
	if err != nil {
		printErr("[ssh %v] unable to parse private key: %v\n", cfg.Server, err)
		return false, err
	}

	hostkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cfg.Hostkey))
	if err != nil {
		printErr("[ssh %v] unable to parse host key: %v\n", cfg.Server, err)
		return false, err
	}

	clientCfg := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(hostkey),
		HostKeyAlgorithms: []string{
			hostkey.Type(),
		},
	}

	verbose("[ssh %v] trying to connect\n", cfg.Server)

	client, err := ssh.Dial("tcp", cfg.Server, clientCfg)
	if err != nil {
		printErr("[ssh %v] error: %v\n", cfg.Server, err)
		return false, err
	}

	verbose("[ssh %v] connected\n", cfg.Server)

	remoteListen, err := client.Listen("tcp", cfg.RemoteListen)
	if err != nil {
		printErr("[ssh %v] unable to listen remotely on %v: %v\n", cfg.Server, cfg.RemoteListen, err)
		_ = client.Close()
		return true, err
	}

	wg := &errgroup.Group{}
	for {
		incoming, err := remoteListen.Accept()
		if err != nil {
			printErr("[ssh %v] error accepting incoming remote connection: %v\n", cfg.Server, err)
			break
		}

		verbose("[ssh %v] new incoming connection from remote %v, err %v\n", cfg.Server, incoming.RemoteAddr(), err)

		conn, err := net.Dial("tcp", target)
		if err != nil {
			printErr("[ssh %v] unable to connect to target %v: %v\n", cfg.Server, target, err)
			_ = client.Close()
			return true, err
		}

		verbose("[ssh %v] success, connected to target %v, start forwarding data\n", cfg.Server, target)
		forward(wg, incoming, conn)
	}

	err = wg.Wait()
	if err != nil {
		_ = client.Close()
		return true, err
	}

	return true, client.Close()
}

// forwardSSH connects to an SSH server and creates a new remote port forward to target.
func forwardSSH(cfg SSHConnection, target string) {
	for {
		success, err := connectSSH(cfg, target)
		if err != nil {
			printErr("[ssh %v] error: %v\n", cfg.Server, err)
		}

		if success {
			time.Sleep(opts.ReconnectDelay)
		} else {
			time.Sleep(opts.BackoffDelay)
		}
	}
}
