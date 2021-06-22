package testutils

import (
	"github.com/Azure/Orkestra/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func BookinfoApplication(targetNamespace string) v1alpha1.Application {
	values := []byte(`{
		"productpage": {
			"replicaCount": 1
		},
		"details": {
			"replicaCount": 1
		},
		"reviews": {
			"replicaCount": 1
		},
		"ratings": {
			"replicaCount": 1
		}
	}`)
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name: Bookinfo,
			Dependencies: []string{
				Ambassador,
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     BookinfoChartURL,
				Name:    Bookinfo,
				Version: BookinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: targetNamespace,
				Values: &apiextensionsv1.JSON{
					Raw: values,
				},
				Interval: defaultDuration,
			},
			Subcharts: []v1alpha1.DAG{
				{
					Name:         "productpage",
					Dependencies: []string{"reviews"},
				},
				{
					Name:         "reviews",
					Dependencies: []string{"details", "ratings"},
				},
				{
					Name:         "ratings",
					Dependencies: []string{},
				},
				{
					Name:         "details",
					Dependencies: []string{},
				},
			},
		},
	}
}
