package templates

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/Azure/Orkestra/pkg/executor"
	"github.com/Azure/Orkestra/pkg/graph"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/google/go-cmp/cmp"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func subChartHelper(values map[string]interface{}, subChartName string) *apiextensionsv1.JSON {
	scValues, _ := graph.SubChartValues(subChartName, values)
	return scValues
}

func Test_GenerateTemplates(t *testing.T) {
	type args struct {
		graph       *graph.Graph
		namespace   string
		parallelism *int64
	}
	relValues := map[string]interface{}{
		"global":     map[string]string{"keyG": "valueG"},
		"subchart-1": map[string]string{"sc1-key": "sc1-value"},
		"subchart-2": map[string]string{"sc2-key": "sc2-value"},
		"subchart-3": map[string]string{"sc3-key": "sc3-value"},
	}
	bytesRelValues, _ := json.Marshal(relValues)
	var p int64 = 0

	tests := []struct {
		name string
		args args
		want []v1alpha13.Template
	}{
		{
			name: "Test Single Application with Multiple Executors",
			args: args{
				graph: &graph.Graph{
					Name: "bookinfo",
					AllExecutors: map[string]executor.Executor{
						executor.HelmReleaseForward{}.GetName(): executor.HelmReleaseForward{},
						executor.KeptnForward{}.GetName():       executor.KeptnForward{},
					},
					Nodes: map[string]*graph.AppNode{
						"ambassador": {
							Name: "ambassador",
							Tasks: map[string]*graph.TaskNode{
								"ambassador-ambassador": {
									Name:         "ambassador-ambassador",
									ChartName:    "ambassador",
									ChartVersion: "1.0.0",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
										"keptn": {
											Name:         "keptn",
											Executor:     executor.KeptnForward{},
											Dependencies: []string{"helmrelease"},
										},
									},
								},
							},
						},
						"bookinfo": {
							Name:         "bookinfo",
							Dependencies: []string{"ambassador"},
							Tasks: map[string]*graph.TaskNode{
								"bookinfo-bookinfo": {
									Name:         "bookinfo-bookinfo",
									ChartName:    "bookinfo",
									ChartVersion: "0.1.6",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
										"keptn": {
											Name:         "keptn",
											Executor:     executor.KeptnForward{},
											Dependencies: []string{"helmrelease"},
										},
									},
								},
							},
						},
					},
				},
				namespace:   "testorkestra",
				parallelism: &p,
			},
			want: []v1alpha13.Template{
				{
					Name:        "bookinfo",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							{
								Name:     "bookinfo-bookinfo",
								Template: "bookinfo-bookinfo",
							},
						},
					},
				},
				{
					Name:        "ambassador",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							{
								Name:     "ambassador-ambassador",
								Template: "ambassador-ambassador",
							},
						},
					},
				},
				{
					Name:        "ambassador-ambassador",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							executor.HelmReleaseForward{}.GetTask("helmrelease", nil, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "ambassador",
										Labels: map[string]string{
											v1alpha1.ChartLabel:     "ambassador",
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "ambassador",
												Version: "1.0.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "ambassador",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
										},
									},
								}),
							),
							executor.KeptnForward{}.GetTask("keptn", []string{"helmrelease"}, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "ambassador",
										Labels: map[string]string{
											v1alpha1.ChartLabel:     "ambassador",
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "ambassador",
												Version: "1.0.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "ambassador",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
										},
									},
								}),
							),
						},
					},
				},
				{
					Name:        "bookinfo-bookinfo",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							executor.HelmReleaseForward{}.GetTask("helmrelease", nil, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "bookinfo",
										Labels: map[string]string{
											v1alpha1.ChartLabel:     "bookinfo",
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "bookinfo",
												Version: "0.1.6",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "bookinfo",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
										},
									},
								}),
							),
							executor.KeptnForward{}.GetTask("keptn", []string{"helmrelease"}, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "bookinfo",
										Labels: map[string]string{
											v1alpha1.ChartLabel:     "bookinfo",
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "bookinfo",
												Version: "0.1.6",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "bookinfo",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
										},
									},
								}),
							),
						},
					},
				},
				executor.HelmReleaseForward{}.GetTemplate(),
				executor.KeptnForward{}.GetTemplate(),
			},
		},
		{
			name: "Test Single Application with Sub-Charts",
			args: args{
				graph: &graph.Graph{
					Name: "bookinfo",
					AllExecutors: map[string]executor.Executor{
						"helmrelease-forward-executor": executor.HelmReleaseForward{},
					},
					Nodes: map[string]*graph.AppNode{
						"ambassador": {
							Name: "ambassador",
							Tasks: map[string]*graph.TaskNode{
								"ambassador-ambassador": {
									Name:         "ambassador-ambassador",
									ChartName:    "ambassador",
									ChartVersion: "1.0.0",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
									},
								},
							},
						},
						"bookinfo": {
							Name:         "bookinfo",
							Dependencies: []string{"ambassador"},
							Tasks: map[string]*graph.TaskNode{
								"bookinfo-bookinfo": {
									Name:         "bookinfo-bookinfo",
									ChartName:    "bookinfo",
									ChartVersion: "0.1.6",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
									Dependencies: []string{"bookinfo-subchart-1", "bookinfo-subchart-2", "bookinfo-subchart-3"},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
									},
								},
								"bookinfo-subchart-1": {
									Name:         "bookinfo-subchart-1",
									ChartName:    utils.GetSubchartName("bookinfo", "subchart-1"),
									ChartVersion: "0.1.0",
									Parent:       "bookinfo",
									Release: &v1alpha1.Release{
										Values: subChartHelper(relValues, "subchart-1"),
									},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
									},
								},
								"bookinfo-subchart-2": {
									Name:         "bookinfo-subchart-2",
									ChartName:    utils.GetSubchartName("bookinfo", "subchart-2"),
									ChartVersion: "0.1.0",
									Parent:       "bookinfo",
									Release: &v1alpha1.Release{
										Values: subChartHelper(relValues, "subchart-2"),
									},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
									},
								},
								"bookinfo-subchart-3": {
									Name:         "bookinfo-subchart-3",
									ChartName:    utils.GetSubchartName("bookinfo", "subchart-3"),
									ChartVersion: "0.1.0",
									Parent:       "bookinfo",
									Release: &v1alpha1.Release{
										Values: subChartHelper(relValues, "subchart-3"),
									},
									Dependencies: []string{"bookinfo-subchart-1", "bookinfo-subchart-2"},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
									},
								},
							},
						},
					},
				},
				namespace:   "testorkestra",
				parallelism: &p,
			},
			want: []v1alpha13.Template{
				{
					Name:        "bookinfo",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							executor.HelmReleaseForward{}.GetTask("bookinfo-bookinfo", []string{"bookinfo-subchart-1", "bookinfo-subchart-2", "bookinfo-subchart-3"}, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "bookinfo",
										Labels: map[string]string{
											v1alpha1.ChartLabel:     "bookinfo",
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "bookinfo",
												Version: "0.1.6",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "bookinfo",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
										},
									},
								}),
							),
							executor.HelmReleaseForward{}.GetTask("bookinfo-subchart-1", nil, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: utils.GetSubchartName("bookinfo", "subchart-1"),
										Labels: map[string]string{
											v1alpha1.ChartLabel:     utils.GetSubchartName("bookinfo", "subchart-1"),
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
										Annotations: map[string]string{
											v1alpha1.ParentChartAnnotation: "bookinfo",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   utils.GetSubchartName("bookinfo", "subchart-1"),
												Version: "0.1.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: utils.GetSubchartName("bookinfo", "subchart-1"),
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"sc1-key":"sc1-value"}`),
										},
									},
								}),
							),
							executor.HelmReleaseForward{}.GetTask("bookinfo-subchart-2", nil, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: utils.GetSubchartName("bookinfo", "subchart-2"),
										Labels: map[string]string{
											v1alpha1.ChartLabel:     utils.GetSubchartName("bookinfo", "subchart-2"),
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
										Annotations: map[string]string{
											v1alpha1.ParentChartAnnotation: "bookinfo",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   utils.GetSubchartName("bookinfo", "subchart-2"),
												Version: "0.1.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: utils.GetSubchartName("bookinfo", "subchart-2"),
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"sc2-key":"sc2-value"}`),
										},
									},
								}),
							),
							executor.HelmReleaseForward{}.GetTask("bookinfo-subchart-3", []string{"bookinfo-subchart-1", "bookinfo-subchart-2"}, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: utils.GetSubchartName("bookinfo", "subchart-3"),
										Labels: map[string]string{
											v1alpha1.ChartLabel:     utils.GetSubchartName("bookinfo", "subchart-3"),
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
										Annotations: map[string]string{
											v1alpha1.ParentChartAnnotation: "bookinfo",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   utils.GetSubchartName("bookinfo", "subchart-3"),
												Version: "0.1.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: utils.GetSubchartName("bookinfo", "subchart-3"),
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"sc3-key":"sc3-value"}`),
										},
									},
								}),
							),
						},
					},
				},
				{
					Name:        "ambassador",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							executor.HelmReleaseForward{}.GetTask("ambassador-ambassador", nil, getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "ambassador",
										Labels: map[string]string{
											v1alpha1.ChartLabel:     "ambassador",
											v1alpha1.OwnershipLabel: "bookinfo",
											v1alpha1.HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "ambassador",
												Version: "1.0.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "ambassador",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
										},
									},
								}),
							),
						},
					},
				},
				executor.HelmReleaseForward{}.GetTemplate(),
			},
		},
		{
			name: "Test Single Application without Sub-chart",
			args: args{
				graph: &graph.Graph{
					Name: "bookinfo",
					AllExecutors: map[string]executor.Executor{
						"helmrelease-forward-executor": executor.HelmReleaseForward{},
					},
					Nodes: map[string]*graph.AppNode{
						"bookinfo": {
							Name: "bookinfo",
							Tasks: map[string]*graph.TaskNode{
								"bookinfo-bookinfo": {
									Name:         "bookinfo-bookinfo",
									ChartName:    "bookinfo",
									ChartVersion: "0.1.6",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
									Executors: map[string]*graph.ExecutorNode{
										"helmrelease": {
											Name:     "helmrelease",
											Executor: executor.HelmReleaseForward{},
										},
									},
								},
							},
						},
					},
				},
				namespace:   "testorkestra",
				parallelism: &p,
			},
			want: []v1alpha13.Template{
				{
					Name:        "bookinfo",
					Parallelism: &p,
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							{
								Name:     "bookinfo-bookinfo",
								Template: "helmrelease-forward-executor",
								Arguments: v1alpha13.Arguments{
									Parameters: []v1alpha13.Parameter{
										{
											Name: "helmrelease",
											Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "bookinfo",
													Labels: map[string]string{
														v1alpha1.ChartLabel:     "bookinfo",
														v1alpha1.OwnershipLabel: "bookinfo",
														v1alpha1.HeritageLabel:  "orkestra",
													},
												},
												Spec: fluxhelmv2beta1.HelmReleaseSpec{
													Chart: fluxhelmv2beta1.HelmChartTemplate{
														Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
															Chart:   "bookinfo",
															Version: "0.1.6",
															SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
																Kind:      "HelmRepository",
																Name:      "chartmuseum",
																Namespace: "testorkestra",
															},
														},
													},
													ReleaseName: "bookinfo",
													Values: &apiextensionsv1.JSON{
														Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
													},
												},
											}),
										},
										{
											Name:  "timeout",
											Value: getTimeout(nil),
										},
									},
								},
							},
						},
					},
				},
				executor.HelmReleaseForward{}.GetTemplate(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := NewTemplateGenerator(tt.args.namespace, tt.args.parallelism)
			tg.GenerateTemplates(tt.args.graph)

			// Sort all the lists so that comparison is consistent
			for _, item := range [][]v1alpha13.Template{tg.Templates, tt.want} {
				for _, tpl := range item {
					if tpl.DAG != nil {
						sort.Slice(tpl.DAG.Tasks, func(i, j int) bool {
							return tpl.DAG.Tasks[i].Name < tpl.DAG.Tasks[j].Name
						})
					}
				}
				sort.Slice(item, func(i, j int) bool {
					return item[i].Name < item[j].Name
				})
			}
			if !cmp.Equal(tg.Templates, tt.want) {
				t.Errorf("GenerateTemplates() = %v", cmp.Diff(tg.Templates, tt.want))
			}
		})
	}
}

func Test_createHelmRelease(t *testing.T) {
	type args struct {
		taskNode    *graph.TaskNode
		graphName   string
		namespace   string
		parallelism *int64
	}
	var p int64 = 0
	tests := []struct {
		name string
		args args
		want *fluxhelmv2beta1.HelmRelease
	}{
		{
			name: "testing with valid release",
			args: args{
				taskNode: &graph.TaskNode{
					Name:         "myAppChart",
					ChartName:    "myAppChart",
					ChartVersion: "0.1.0",
					Release: &v1alpha1.Release{
						TargetNamespace: "targetOrkestra",
					},
				},
				graphName:   "mygraph",
				parallelism: &p,
				namespace:   "testorkestra",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      "myappchart",
					Namespace: "targetOrkestra",
					Labels: map[string]string{
						v1alpha1.ChartLabel:     "myAppChart",
						v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
						v1alpha1.OwnershipLabel: "mygraph",
					},
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   "myappchart",
							Version: "0.1.0",
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: "testorkestra",
							},
						},
					},
					ReleaseName:     "myappchart",
					TargetNamespace: "targetOrkestra",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := NewTemplateGenerator(tt.args.namespace, tt.args.parallelism)
			got := tg.createHelmRelease(tt.args.taskNode, tt.args.graphName)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("createHelmRelease() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}
