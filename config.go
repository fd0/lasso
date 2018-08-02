package main

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclparse"
)

// Config is parsed from a configuration file.
type Config struct {
	Target         string `hcl:"target"`
	ConnectTimeout int    `hcl:"connect_timeout,optional"`
	ReconnectDelay int    `hcl:"reconnect_delay,optional"`
	BackoffDelay   int    `hcl:"backoff_delay,optional"`

	TCP []TCPConnection `hcl:"tcp,block"`
	SSH []SSHConnection `hcl:"ssh,block"`
}

// DefaultConfig collects default config items.
var DefaultConfig = Config{
	ConnectTimeout: 60,
	ReconnectDelay: 2,
	BackoffDelay:   30,
}

// ParseConfig returns a config from a file.
func ParseConfig(filename string) (Config, error) {
	var cfg = DefaultConfig

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filename)

	if len(diags) != 0 {
		return Config{}, diags
	}

	decodeDiags := gohcl.DecodeBody(file.Body, nil, &cfg)
	diags = append(diags, decodeDiags...)
	if diags.HasErrors() {
		return Config{}, diags
	}

	return cfg, nil
}
