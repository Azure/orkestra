package testutils

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	bookinfo   = "bookinfo"
	ambassador = "ambassador"
	podinfo    = "podinfo"

	ambassadorChartURL        = "https://nitishm.github.io/charts"
	ambassadorOldChartVersion = "6.6.0"
	ambassadorChartVersion    = "6.7.9"

	bookinfoChartURL     = "https://nitishm.github.io/charts"
	bookinfoChartVersion = "v2"

	podinfoChartURL     = "https://stefanprodan.github.io/podinfo"
	podinfoChartVersion = "5.2.1"

	portForwardStagingRepoURL = "http://127.0.0.1:8080"
	inClusterstagingRepoURL   = "http://orkestra-chartmuseum.orkestra:8080"
)

var (
	defaultDuration = metav1.Duration{Duration: time.Minute * 5}     // treat as const
	letterRunes     = []rune("abcdefghijklmnopqrstuvwxyz1234567890") // treat as const
)
