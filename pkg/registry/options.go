package registry

import (
	"os"

	"helm.sh/helm/v3/pkg/action"
)

const (
	ReadWritePerm = 0777
)

type Option func(c *Client)

type PullOption func(p *action.Pull)

// Client Options

func TargetDir(d string) Option {
	// check if target dir exists.
	// if doesnt exist create one.
	if _, err := os.Stat(d); os.IsNotExist(err) {
		_ = os.Mkdir(d, ReadWritePerm)
	}

	return func(c *Client) {
		c.TargetDir = d
	}
}
