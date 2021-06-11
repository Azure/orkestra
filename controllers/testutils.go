package controllers

import (
	"fmt"
	"math/rand"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	bookinfo   = "bookinfo"
	ambassador = "ambassador"
	podinfo    = "podinfo"

	ambassadorChartURL        = "https://www.getambassador.io/helm"
	ambassadorOldChartVersion = "6.6.0"
	ambassadorChartVersion    = "6.7.9"

	bookinfoChartURL     = "https://nitishm.github.io/charts"
	bookinfoChartVersion = "v2"

	podinfoChartURL     = "https://stefanprodan.github.io/podinfo"
	podinfoChartVersion = "5.2.1"
)

var (
	defaultDuration = metav1.Duration{Duration: time.Minute * 5}
)

func defaultAppGroup(targetNamespace string) *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: targetNamespace,
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, bookinfoApplication(targetNamespace), ambassadorApplication(targetNamespace))
	return g
}

func smallAppGroup(targetNamespace string) *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: targetNamespace,
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, podinfoApplication(targetNamespace))
	return g
}

func ambassadorApplication(targetNamespace string) v1alpha1.Application {
	values := []byte(fmt.Sprintf(`{
       "nameOverride": "%s",
	   "service": {
		  "type": "ClusterIP"
	   },
       "scope": {
          "singleNamespace": true
       }
	}`, targetNamespace))
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name: ambassador,
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     ambassadorChartURL,
				Name:    ambassador,
				Version: ambassadorChartVersion,
			},
			Release: &v1alpha1.Release{
				Timeout:         &metav1.Duration{Duration: time.Minute * 10},
				TargetNamespace: targetNamespace,
				Values: &apiextensionsv1.JSON{
					Raw: values,
				},
				Interval: defaultDuration,
			},
		},
	}
}

func bookinfoApplication(targetNamespace string) v1alpha1.Application {
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
			Name: bookinfo,
			Dependencies: []string{
				ambassador,
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     bookinfoChartURL,
				Name:    bookinfo,
				Version: bookinfoChartVersion,
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

func podinfoApplication(targetNamespace string) v1alpha1.Application {
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name:         podinfo,
			Dependencies: []string{},
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     podinfoChartURL,
				Name:    podinfo,
				Version: podinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: targetNamespace,
				Interval:        defaultDuration,
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
