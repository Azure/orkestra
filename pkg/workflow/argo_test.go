package workflow

import (
	"encoding/json"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func helmValues(v string) map[string]interface{} {
	out := make(map[string]interface{})
	err := json.Unmarshal([]byte(v), &out)
	if err != nil {
		panic(err)
	}
	return out
}

func Test_subchartValues(t *testing.T) {
	type args struct {
		sc string
		av helmopv1.HelmValues
	}
	tests := []struct {
		name string
		args args
		want helmopv1.HelmValues
	}{
		{
			name: "withGlobalSuchart",
			args: args{
				sc: "subchart",
				av: helmopv1.HelmValues{
					Data: helmValues(`{"global": {"keyG": "valueG"},"subchart": {"keySC": "valueSC"}}`),
				},
			},
			want: helmopv1.HelmValues{
				Data: helmValues(`{"global": {"keyG": "valueG"},"keySC": "valueSC"}`),
			},
		},
		{
			name: "withOnlyGlobal",
			args: args{
				sc: "subchart",
				av: helmopv1.HelmValues{
					Data: helmValues(`{"global": {"keyG": "valueG"}}`),
				},
			},
			want: helmopv1.HelmValues{
				Data: helmValues(`{"global": {"keyG": "valueG"}}`),
			},
		},
		{
			name: "withOnlySubchart",
			args: args{
				sc: "subchart",
				av: helmopv1.HelmValues{
					Data: helmValues(`{"subchart": {"keySC": "valueSC"}}`),
				},
			},
			want: helmopv1.HelmValues{
				Data: helmValues(`{"keySC": "valueSC"}`),
			},
		},
		{
			name: "withNone",
			args: args{
				sc: "subchart",
				av: helmopv1.HelmValues{
					Data: make(map[string]interface{}),
				},
			},
			want: helmopv1.HelmValues{
				Data: make(map[string]interface{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subchartValues(tt.args.sc, tt.args.av); !cmp.Equal(got, tt.want) {
				t.Errorf("subchartValues() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_generateSubchartAndAppDAGTasks(t *testing.T) {
	type args struct {
		app      *v1alpha1.Application
		targetNS string
	}
	tests := []struct {
		name    string
		args    args
		want    []v1alpha12.DAGTask
		wantErr bool
	}{
		{
			name: "sequential",
			args: args{
				app: &v1alpha1.Application{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationSpec{
						Subcharts: []v1alpha1.DAG{
							{
								Name:         "subchart-3",
								Dependencies: []string{"subchart-2"},
							},
							{
								Name:         "subchart-2",
								Dependencies: []string{"subchart-1"},
							},
							{
								Name:         "subchart-1",
								Dependencies: nil,
							},
						},
						HelmReleaseSpec: helmopv1.HelmReleaseSpec{
							Values: helmopv1.HelmValues{
								Data: map[string]interface{}{
									"global": map[string]interface{}{
										"keyG": "valueG",
									},
									"subchart-1": map[string]interface{}{
										"subchart-1-key": "subchart-1-value",
									},
									"subchart-2": map[string]interface{}{
										"subchart-2-key": "subchart-2-value",
									},
									"subchart-3": map[string]interface{}{
										"subchart-3-key": "subchart-3-value",
									},
								},
							},
							ChartSource: helmopv1.ChartSource{
								RepoChartSource: &helmopv1.RepoChartSource{
									RepoURL: "http://stagingrepo",
									Name:    "appchart",
									Version: "1.0.0",
								},
							},
						},
					},
				},
			},
			want: []v1alpha12.DAGTask{
				{
					Name:         "subchart-3",
					Template:     "helmrelease-executor",
					Dependencies: []string{"subchart-2"},
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "subchart-3",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "subchart-3",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-3-key": "subchart-3-value",
											},
										},
									},
								})),
							},
						},
					},
				},
				{
					Name:         "subchart-2",
					Template:     "helmrelease-executor",
					Dependencies: []string{"subchart-1"},
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "subchart-2",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "subchart-2",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-2-key": "subchart-2-value",
											},
										},
									},
								})),
							},
						},
					},
				},
				{
					Name:     "subchart-1",
					Template: "helmrelease-executor",
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "subchart-1",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "subchart-1",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-1-key": "subchart-1-value",
											},
										},
									},
								})),
							},
						},
					},
				},
				{
					Name:         "application",
					Template:     "helmrelease-executor",
					Dependencies: []string{"subchart-3", "subchart-2", "subchart-1"},
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "application",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "appchart",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-1": map[string]interface{}{
													"subchart-1-key": "subchart-1-value",
												},
												"subchart-2": map[string]interface{}{
													"subchart-2-key": "subchart-2-value",
												},
												"subchart-3": map[string]interface{}{
													"subchart-3-key": "subchart-3-value",
												},
											},
										},
									},
								})),
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "parallel",
			args: args{
				app: &v1alpha1.Application{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationSpec{
						Subcharts: []v1alpha1.DAG{
							{
								Name:         "subchart-3",
								Dependencies: []string{"subchart-2", "subchart-1"},
							},
							{
								Name:         "subchart-2",
								Dependencies: nil,
							},
							{
								Name:         "subchart-1",
								Dependencies: nil,
							},
						},
						HelmReleaseSpec: helmopv1.HelmReleaseSpec{
							Values: helmopv1.HelmValues{
								Data: map[string]interface{}{
									"global": map[string]interface{}{
										"keyG": "valueG",
									},
									"subchart-1": map[string]interface{}{
										"subchart-1-key": "subchart-1-value",
									},
									"subchart-2": map[string]interface{}{
										"subchart-2-key": "subchart-2-value",
									},
									"subchart-3": map[string]interface{}{
										"subchart-3-key": "subchart-3-value",
									},
								},
							},
							ChartSource: helmopv1.ChartSource{
								RepoChartSource: &helmopv1.RepoChartSource{
									RepoURL: "http://stagingrepo",
									Name:    "appchart",
									Version: "1.0.0",
								},
							},
						},
					},
				},
			},
			want: []v1alpha12.DAGTask{
				{
					Name:         "subchart-3",
					Template:     "helmrelease-executor",
					Dependencies: []string{"subchart-2", "subchart-1"},
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "subchart-3",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "subchart-3",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-3-key": "subchart-3-value",
											},
										},
									},
								})),
							},
						},
					},
				},
				{
					Name:     "subchart-2",
					Template: "helmrelease-executor",
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "subchart-2",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "subchart-2",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-2-key": "subchart-2-value",
											},
										},
									},
								})),
							},
						},
					},
				},
				{
					Name:     "subchart-1",
					Template: "helmrelease-executor",
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "subchart-1",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "subchart-1",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-1-key": "subchart-1-value",
											},
										},
									},
								})),
							},
						},
					},
				},
				{
					Name:         "application",
					Template:     "helmrelease-executor",
					Dependencies: []string{"subchart-3", "subchart-2", "subchart-1"},
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							{
								Name: helmReleaseArg,
								Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.fluxcd.io/v1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "application",
									},
									Spec: helmopv1.HelmReleaseSpec{
										ChartSource: helmopv1.ChartSource{
											RepoChartSource: &helmopv1.RepoChartSource{
												RepoURL: "http://stagingrepo",
												Name:    "appchart",
												Version: "1.0.0",
											},
										},
										Values: helmopv1.HelmValues{
											Data: map[string]interface{}{
												"global": map[string]interface{}{
													"keyG": "valueG",
												},
												"subchart-1": map[string]interface{}{
													"subchart-1-key": "subchart-1-value",
												},
												"subchart-2": map[string]interface{}{
													"subchart-2-key": "subchart-2-value",
												},
												"subchart-3": map[string]interface{}{
													"subchart-3-key": "subchart-3-value",
												},
											},
										},
									},
								})),
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateSubchartAndAppDAGTasks(tt.args.app, "http://stagingrepo", tt.args.targetNS)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateSubchartDAGTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("generateSubchartDAGTasks() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_generateAppDAGTemplates(t *testing.T) {
	type args struct {
		apps []*v1alpha1.Application
		repo string
	}
	tests := []struct {
		name    string
		args    args
		want    []v1alpha12.Template
		wantErr bool
	}{
		{
			name: "singleApplicationWithSubchartDAG",
			args: args{
				apps: []*v1alpha1.Application{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "application",
						},
						Spec: v1alpha1.ApplicationSpec{
							HelmReleaseSpec: helmopv1.HelmReleaseSpec{
								Values: helmopv1.HelmValues{
									Data: map[string]interface{}{
										"global": map[string]interface{}{
											"keyG": "valueG",
										},
									},
								},
								ChartSource: helmopv1.ChartSource{
									RepoChartSource: &helmopv1.RepoChartSource{
										RepoURL: "http://primaryrepo",
										Name:    "appchart",
										Version: "1.0.0",
									},
								},
							},
							Subcharts: []v1alpha1.DAG{
								{
									Name:         "subchart-3",
									Dependencies: []string{"subchart-2", "subchart-1"},
								},
								{
									Name:         "subchart-2",
									Dependencies: nil,
								},
								{
									Name:         "subchart-1",
									Dependencies: nil,
								},
							},
						},
					},
				},
				repo: "http://stagingrepo",
			},
			want: []v1alpha12.Template{
				{
					Name: "application",
					DAG: &v1alpha12.DAGTemplate{
						Tasks: []v1alpha12.DAGTask{
							{
								Name:         "subchart-3",
								Dependencies: []string{"subchart-2", "subchart-1"},
								Template:     "helmrelease-executor",
								Arguments: v1alpha12.Arguments{
									Parameters: []v1alpha12.Parameter{
										{
											Name: helmReleaseArg,
											Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.fluxcd.io/v1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "subchart-3",
												},
												Spec: helmopv1.HelmReleaseSpec{
													ChartSource: helmopv1.ChartSource{
														RepoChartSource: &helmopv1.RepoChartSource{
															RepoURL: "http://stagingrepo",
															Name:    "subchart-3",
															Version: "1.0.0",
														},
													},
													Values: helmopv1.HelmValues{
														Data: map[string]interface{}{
															"global": map[string]interface{}{
																"keyG": "valueG",
															},
														},
													},
												},
											})),
										},
									},
								},
							},
							{
								Name:     "subchart-2",
								Template: "helmrelease-executor",
								Arguments: v1alpha12.Arguments{
									Parameters: []v1alpha12.Parameter{
										{
											Name: helmReleaseArg,
											Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.fluxcd.io/v1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "subchart-2",
												},
												Spec: helmopv1.HelmReleaseSpec{
													ChartSource: helmopv1.ChartSource{
														RepoChartSource: &helmopv1.RepoChartSource{
															RepoURL: "http://stagingrepo",
															Name:    "subchart-2",
															Version: "1.0.0",
														},
													},
													Values: helmopv1.HelmValues{
														Data: map[string]interface{}{
															"global": map[string]interface{}{
																"keyG": "valueG",
															},
														},
													},
												},
											})),
										},
									},
								},
							},
							{
								Name:     "subchart-1",
								Template: "helmrelease-executor",
								Arguments: v1alpha12.Arguments{
									Parameters: []v1alpha12.Parameter{
										{
											Name: helmReleaseArg,
											Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.fluxcd.io/v1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "subchart-1",
												},
												Spec: helmopv1.HelmReleaseSpec{
													ChartSource: helmopv1.ChartSource{
														RepoChartSource: &helmopv1.RepoChartSource{
															RepoURL: "http://stagingrepo",
															Name:    "subchart-1",
															Version: "1.0.0",
														},
													},
													Values: helmopv1.HelmValues{
														Data: map[string]interface{}{
															"global": map[string]interface{}{
																"keyG": "valueG",
															},
														},
													},
												},
											})),
										},
									},
								},
							},
							{
								Name:         "application",
								Template:     "helmrelease-executor",
								Dependencies: []string{"subchart-3", "subchart-2", "subchart-1"},
								Arguments: v1alpha12.Arguments{
									Parameters: []v1alpha12.Parameter{
										{
											Name: helmReleaseArg,
											Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.fluxcd.io/v1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "application",
												},
												Spec: helmopv1.HelmReleaseSpec{
													ChartSource: helmopv1.ChartSource{
														RepoChartSource: &helmopv1.RepoChartSource{
															RepoURL: "http://stagingrepo",
															Name:    "appchart",
															Version: "1.0.0",
														},
													},
													Values: helmopv1.HelmValues{
														Data: map[string]interface{}{
															"global": map[string]interface{}{
																"keyG": "valueG",
															},
														},
													},
												},
											})),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "singleApplicationWithNoSubchartDAG",
			args: args{
				apps: []*v1alpha1.Application{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "application",
						},
						Spec: v1alpha1.ApplicationSpec{
							HelmReleaseSpec: helmopv1.HelmReleaseSpec{
								Values: helmopv1.HelmValues{
									Data: map[string]interface{}{
										"global": map[string]interface{}{
											"keyG": "valueG",
										},
									},
								},
								ChartSource: helmopv1.ChartSource{
									RepoChartSource: &helmopv1.RepoChartSource{
										RepoURL: "http://primaryrepo",
										Name:    "appchart",
										Version: "1.0.0",
									},
								},
							},
						},
					},
				},
				repo: "http://stagingrepo",
			},
			want: []v1alpha12.Template{
				{
					Name: "application",
					DAG: &v1alpha12.DAGTemplate{
						Tasks: []v1alpha12.DAGTask{
							{
								Name:     "application",
								Template: "helmrelease-executor",
								Arguments: v1alpha12.Arguments{
									Parameters: []v1alpha12.Parameter{
										{
											Name: helmReleaseArg,
											Value: strToStrPtr(hrToYAML(helmopv1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.fluxcd.io/v1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "application",
												},
												Spec: helmopv1.HelmReleaseSpec{
													Values: helmopv1.HelmValues{
														Data: map[string]interface{}{
															"global": map[string]interface{}{
																"keyG": "valueG",
															},
														},
													},
													ChartSource: helmopv1.ChartSource{
														RepoChartSource: &helmopv1.RepoChartSource{
															RepoURL: "http://primaryrepo",
															Name:    "appchart",
															Version: "1.0.0",
														},
													},
												},
											})),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateAppDAGTemplates(tt.args.apps, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateAppDAGTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("generateAppDAGTemplates() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_argo_generateAppGroupTpls(t *testing.T) {
	type fields struct {
		scheme         *runtime.Scheme
		cli            client.Client
		wf             *v1alpha12.Workflow
		stagingRepoURL string
	}
	type args struct {
		g    *v1alpha1.ApplicationGroup
		apps []*v1alpha1.Application
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &argo{
				scheme:         tt.fields.scheme,
				cli:            tt.fields.cli,
				wf:             tt.fields.wf,
				stagingRepoURL: tt.fields.stagingRepoURL,
			}
			if err := a.generateAppGroupTpls(tt.args.g, tt.args.apps); (err != nil) != tt.wantErr {
				t.Errorf("argo.generateAppGroupTpls() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
