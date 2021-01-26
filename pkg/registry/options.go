package registry

import (
	"github.com/chartmuseum/helm-push/pkg/chartmuseum"
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

// TODO (nitishm) : Add more options to connect with the repositories

// Pull Options

func WithPullOpts(opts ...PullOption) Option {
	return func(c *Client) {
		for _, opt := range opts {
			opt(c.cfg.pull)
		}
	}
}

// Push Options

func WithPushOpts(opts ...chartmuseum.Option) Option {
	return func(c *Client) {
		var err error
		c.cfg.push, err = chartmuseum.NewClient(opts...)
		if err != nil {
			c.cfg.push = &chartmuseum.Client{}
		}
	}
}
