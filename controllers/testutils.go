package controllers

import (
	"math/rand"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Bookinfo   = "bookinfo"
	Ambassador = "ambassador"
	Podinfo    = "podinfo"

	AmbassadorChartURL     = "https://www.getambassador.io/helm"
	AmbassadorChartVersion = "6.6.0"

	BookinfoChartURL     = "https://nitishm.github.io/charts"
	BookinfoChartVersion = "v1"

	PodinfoChartURL     = "https://stefanprodan.github.io/podinfo"
	PodinfoChartVersion = "5.2.1"
)

func bookinfo() *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: "bookinfo",
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, bookinfoApplication(), ambassadorApplication())
	return g
}

func ambassadorApplication() v1alpha1.Application {
	values := []byte(`{
	   "service": {
		  "type": "ClusterIP"
	   }
	}`)
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name: Ambassador,
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				Url:     AmbassadorChartURL,
				Name:    Ambassador,
				Version: AmbassadorChartVersion,
			},
			Release: &v1alpha1.Release{
				Timeout:         &metav1.Duration{Duration: time.Minute * 10},
				TargetNamespace: Ambassador,
				Values: &apiextensionsv1.JSON{
					Raw: values,
				},
			},
		},
	}
}

func bookinfoApplication() v1alpha1.Application {
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
				Url:     BookinfoChartURL,
				Name:    Bookinfo,
				Version: BookinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: Bookinfo,
				Values: &apiextensionsv1.JSON{
					Raw: values,
				},
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

func podinfoApplication() v1alpha1.Application {
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name: Podinfo,
			Dependencies: []string{
				Ambassador,
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				Url:     PodinfoChartURL,
				Name:    Podinfo,
				Version: PodinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: Podinfo,
			},
		},
	}
}

func AddApplication(appGroup v1alpha1.ApplicationGroup, app v1alpha1.Application) v1alpha1.ApplicationGroup {
	appGroup.Spec.Applications = append(appGroup.Spec.Applications, app)
	return appGroup
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
