package testutils

import (
	"github.com/Azure/Orkestra/api/v1alpha1"
)

func PodinfoApplication(targetNamespace string) v1alpha1.Application {
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name:         Podinfo,
			Dependencies: []string{},
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     PodinfoChartURL,
				Name:    Podinfo,
				Version: PodinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: targetNamespace,
				Interval:        defaultDuration,
			},
		},
	}
}
