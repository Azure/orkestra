package testutils

import (
	"fmt"
	"time"

	"github.com/Azure/Orkestra/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AmbassadorApplication(targetNamespace string) v1alpha1.Application {
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
			Name: Ambassador,
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     AmbassadorChartURL,
				Name:    Ambassador,
				Version: AmbassadorChartVersion,
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
