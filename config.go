package main

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclparse"
)

// Config is parsed from a configuration file.
type Config struct {
	Target         string `hcl:"target"`
	ReconnectDelay int    `hcl:"reconnect_delay,optional"`
	BackoffDelay   int    `hcl:"backoff_delay,optional"`

	TCP []TCPConnection `hcl:"tcp,block"`
	SSH []SSHConnection `hcl:"ssh,block"`
}

// DefaultConfig collects default config items.
var DefaultConfig = Config{
	ReconnectDelay: 2,
	BackoffDelay:   30,
}

// TCPConnection describes one plain tcp connection to a server.
type TCPConnection struct {
	Server string `hcl:"server"`
}

// SSHConnection describes one SSH connection to a server.
type SSHConnection struct {
	Server       string `hcl:"server"`
	RemoteListen string `hcl:"remote_listen"`
	User         string `hcl:"user"`
	Hostkey      string `hcl:"hostkey"`
	Key          string `hcl:"key"`
}

// ParseConfig returns a config from a file.
func ParseConfig(filename string) (Config, error) {
	var cfg = DefaultConfig

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filename)

	decodeDiags := gohcl.DecodeBody(file.Body, nil, &cfg)
	diags = append(diags, decodeDiags...)
	if diags.HasErrors() {
		return Config{}, diags
	}

	return cfg, nil
}
