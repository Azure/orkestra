package pkg

import (
	"net"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
)

type Probe struct {
	health healthcheck.Handler
}

func ProbeHandler(chartmuseumAddress, endpoint string) (*Probe, error) {
	health := healthcheck.NewHandler()
	// Liveness check verifies that the number of goroutines are below threshold
	health.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(100))
	// Readiness check verifies that chartmuseum is up and serving traffic
	health.AddReadinessCheck("chartmuseum-ready", healthcheck.HTTPGetCheck(chartmuseumAddress+"/"+endpoint, time.Minute))

	return &Probe{
		health: health,
	}, nil
}

func (p *Probe) Start(port string) {
	go http.ListenAndServe(net.JoinHostPort("0.0.0.0", port), p.health)
}
