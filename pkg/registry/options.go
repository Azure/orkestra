package registry

import (
	"helm.sh/helm/v3/pkg/action"
)

type Option func(c *Client)

type PullOption func(p *action.Pull)

// Client Options

func TargetDir(d string) Option {
	return func(c *Client) {
		c.TargetDir = d
	}
}
