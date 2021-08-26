package workflow

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/google/go-cmp/cmp"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func subChartHelper(values map[string]interface{}, subChartName string) *apiextensionsv1.JSON {
	scValues, _ := subChartValues(subChartName, values)
	return scValues
}

func Test_generateAppDAGTemplates(t *testing.T) {
	type args struct {
		graph       *Graph
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
		name    string
		args    args
		want    map[string]v1alpha13.Template
		wantErr bool
	}{
		{
			name: "testing singleApplicationWithSubchartDAG",
			args: args{
				graph: &Graph{
					Name: "bookinfo",
					Nodes: map[string]*AppNode{
						"ambassador": {
							Name: "ambassador",
							Tasks: map[string]*TaskNode{
								"ambassador": {
									Name:         "ambassador",
									ChartName:    "ambassador",
									ChartVersion: "1.0.0",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
								},
							},
						},
						"bookinfo": {
							Name:         "bookinfo",
							Dependencies: []string{"ambassador"},
							Tasks: map[string]*TaskNode{
								"bookinfo": {
									Name:         "bookinfo",
									ChartName:    "bookinfo",
									ChartVersion: "0.1.6",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
								},
								"subchart-1": {
									Name:         "subchart-1",
									ChartName:    utils.GetSubchartName("bookinfo", "subchart-1"),
									ChartVersion: "0.1.0",
									Parent:       "bookinfo",
									Release: &v1alpha1.Release{
										Values: subChartHelper(relValues, "subchart-1"),
									},
								},
								"subchart-2": {
									Name:         "subchart-2",
									ChartName:    utils.GetSubchartName("bookinfo", "subchart-2"),
									ChartVersion: "0.1.0",
									Parent:       "bookinfo",
									Release: &v1alpha1.Release{
										Values: subChartHelper(relValues, "subchart-2"),
									},
								},
								"subchart-3": {
									Name:         "subchart-3",
									ChartName:    utils.GetSubchartName("bookinfo", "subchart-3"),
									ChartVersion: "0.1.0",
									Parent:       "bookinfo",
									Release: &v1alpha1.Release{
										Values: subChartHelper(relValues, "subchart-3"),
									},
									Dependencies: []string{"subchart-1", "subchart-2"},
								},
							},
						},
					},
				},
				namespace:   "testorkestra",
				parallelism: &p,
			},
			want: map[string]v1alpha13.Template{
				"bookinfo": {
					Name: "bookinfo",
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							appDAGTaskBuilder("bookinfo", getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "bookinfo",
										Labels: map[string]string{
											ChartLabelKey:  "bookinfo",
											OwnershipLabel: "bookinfo",
											HeritageLabel:  "orkestra",
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
							appDAGTaskBuilder("subchart-1", getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: utils.GetSubchartName("bookinfo", "subchart-1"),
										Labels: map[string]string{
											ChartLabelKey:  utils.GetSubchartName("bookinfo", "subchart-1"),
											OwnershipLabel: "bookinfo",
											HeritageLabel:  "orkestra",
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
							appDAGTaskBuilder("subchart-2", getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: utils.GetSubchartName("bookinfo", "subchart-2"),
										Labels: map[string]string{
											ChartLabelKey:  utils.GetSubchartName("bookinfo", "subchart-2"),
											OwnershipLabel: "bookinfo",
											HeritageLabel:  "orkestra",
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
							appDAGTaskBuilder("subchart-3", getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: utils.GetSubchartName("bookinfo", "subchart-3"),
										Labels: map[string]string{
											ChartLabelKey:  utils.GetSubchartName("bookinfo", "subchart-3"),
											OwnershipLabel: "bookinfo",
											HeritageLabel:  "orkestra",
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
					Parallelism: &p,
				},
				"ambassador": {
					Name: "ambassador",
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							appDAGTaskBuilder("ambassador", getTimeout(nil), utils.HrToB64AnyStringPtr(
								&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "ambassador",
										Labels: map[string]string{
											ChartLabelKey:  "ambassador",
											OwnershipLabel: "bookinfo",
											HeritageLabel:  "orkestra",
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
					Parallelism: &p,
				},
			},
			wantErr: false,
		},
		{
			name: "testing singleApplicationWithoutSubchartDAG",
			args: args{
				graph: &Graph{
					Name: "bookinfo",
					Nodes: map[string]*AppNode{
						"bookinfo": {
							Name: "bookinfo",
							Tasks: map[string]*TaskNode{
								"bookinfo": {
									Name:         "bookinfo",
									ChartName:    "bookinfo",
									ChartVersion: "0.1.6",
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
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
			want: map[string]v1alpha13.Template{
				"bookinfo": {
					Name: "bookinfo",
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							{
								Name:     "bookinfo",
								Template: "helmrelease-executor",
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
														ChartLabelKey:  "bookinfo",
														OwnershipLabel: "bookinfo",
														HeritageLabel:  "orkestra",
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
					Parallelism: &p,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateAppDAGTemplates(tt.args.graph, tt.args.namespace, tt.args.parallelism)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateAppDAGTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, tpl := range got {
				sort.Slice(tpl.DAG.Tasks, func(i, j int) bool {
					return tpl.DAG.Tasks[i].Name < tpl.DAG.Tasks[j].Name
				})
			}
			for _, tpl := range tt.want {
				sort.Slice(tpl.DAG.Tasks, func(i, j int) bool {
					return tpl.DAG.Tasks[i].Name < tpl.DAG.Tasks[j].Name
				})
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("generateAppDAGTemplates() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_appDAGTaskBuilder(t *testing.T) {
	type args struct {
		name    string
		timeout *v1alpha13.AnyString
		hrStr   *v1alpha13.AnyString
	}
	tests := []struct {
		name string
		args args
		want v1alpha13.DAGTask
	}{
		{
			name: "testing with nil pointer args",
			args: args{
				name:    "myApp",
				timeout: nil,
				hrStr:   nil,
			},
			want: v1alpha13.DAGTask{
				Name:     "myapp",
				Template: "helmrelease-executor",
				Arguments: v1alpha13.Arguments{
					Parameters: []v1alpha13.Parameter{
						{
							Name:  "helmrelease",
							Value: nil,
						},
						{
							Name:  "timeout",
							Value: nil,
						},
					},
				},
			},
		},
		{
			name: "testing with valid args",
			args: args{
				name:    "myApp",
				timeout: utils.ToAnyStringPtr("5m"),
				hrStr:   utils.ToAnyStringPtr("empty"),
			},
			want: v1alpha13.DAGTask{
				Name:     "myapp",
				Template: "helmrelease-executor",
				Arguments: v1alpha13.Arguments{
					Parameters: []v1alpha13.Parameter{
						{
							Name:  "helmrelease",
							Value: utils.ToAnyStringPtr("empty"),
						},
						{
							Name:  "timeout",
							Value: utils.ToAnyStringPtr("5m"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appDAGTaskBuilder(tt.args.name, tt.args.timeout, tt.args.hrStr)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("appDAGTaskBuilder() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_helmReleaseBuilder(t *testing.T) {
	type args struct {
		r         *v1alpha1.Release
		namespace string
		name      string
		version   string
	}
	tests := []struct {
		name string
		args args
		want *fluxhelmv2beta1.HelmRelease
	}{
		{
			name: "testing with valid release",
			args: args{
				r: &v1alpha1.Release{
					TargetNamespace: "targetOrkestra",
				},
				namespace: "testorkestra",
				name:      "myAppChart",
				version:   "0.1.0",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      "myappchart",
					Namespace: "targetOrkestra",
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
			got := createHelmRelease(tt.args.r, tt.args.namespace, tt.args.name, tt.args.version)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("helmReleaseBuilder() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}
