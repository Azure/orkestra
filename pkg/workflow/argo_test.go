package workflow

import (
	"encoding/json"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func Test_generateSubchartDAGTasks(t *testing.T) {
	type args struct {
		app *v1alpha1.Application
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
				v1alpha12.DAGTask{
					Name: "subchart-1",
					Arguments: v1alpha12.Arguments{
						Parameters: []v1alpha12.Parameter{
							v1alpha12.Parameter{
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
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateSubchartDAGTasks(tt.args.app, "http://stagingrepo")
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
