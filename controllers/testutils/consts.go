package testutils

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Bookinfo   = "bookinfo"
	Ambassador = "ambassador"
	Podinfo    = "podinfo"

	AmbassadorChartURL        = "https://nitishm.github.io/charts"
	AmbassadorOldChartVersion = "6.6.0"
	AmbassadorChartVersion    = "6.7.9"

	BookinfoChartURL     = "https://nitishm.github.io/charts"
	BookinfoChartVersion = "v2"

	PodinfoChartURL     = "https://stefanprodan.github.io/podinfo"
	PodinfoChartVersion = "5.2.1"
)

var (
	defaultDuration = metav1.Duration{Duration: time.Minute * 5}     // treat as const
	letterRunes     = []rune("abcdefghijklmnopqrstuvwxyz1234567890") // treat as const
)
