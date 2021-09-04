package graph

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/executor"
	"github.com/Azure/Orkestra/pkg/utils"
	"github.com/google/go-cmp/cmp"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewForwardGraph(t *testing.T) {
	type args struct {
		appGroup *v1alpha1.ApplicationGroup
	}
	tests := []struct {
		name string
		args args
		want *Graph
	}{
		{
			name: "Basic Ordered Set of Applications",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "application1",
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application1",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application2",
									Dependencies: []string{"application1"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application2",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application3",
									Dependencies: []string{"application2"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application3",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "application",
				AllExecutors: map[string]executor.Executor{
					executor.HelmReleaseForward{}.GetName(): executor.HelmReleaseForward{},
				},
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1-application1": {
								Name:         "application1-application1",
								ChartName:    "application1",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
						},
					},
					"application2": {
						Name:         "application2",
						Dependencies: []string{"application1"},
						Tasks: map[string]*TaskNode{
							"application2-application2": {
								Name:         "application2-application2",
								ChartName:    "application2",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
						},
					},
					"application3": {
						Name:         "application3",
						Dependencies: []string{"application2"},
						Tasks: map[string]*TaskNode{
							"application3-application3": {
								Name:         "application3-application3",
								ChartName:    "application3",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
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
		},
		{
			name: "Applications with Subcharts",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "application1",
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application1",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
									Subcharts: []v1alpha1.DAG{
										{
											Name:         "subchart1",
											Dependencies: []string{"subchart2"},
										},
										{
											Name: "subchart2",
										},
										{
											Name:         "subchart3",
											Dependencies: []string{"subchart1", "subchart2"},
										},
									},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application2",
									Dependencies: []string{"application1"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application2",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
									Subcharts: []v1alpha1.DAG{
										{
											Name: "subchart1",
										},
									},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application3",
									Dependencies: []string{"application2"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application3",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
						},
					},
					Status: v1alpha1.ApplicationGroupStatus{
						Applications: []v1alpha1.ApplicationStatus{
							{
								Subcharts: map[string]v1alpha1.ChartStatus{
									"subchart1": {
										Version: "0.1.0",
									},
									"subchart2": {
										Version: "0.1.0",
									},
									"subchart3": {
										Version: "0.1.0",
									},
								},
							},
							{
								Subcharts: map[string]v1alpha1.ChartStatus{
									"subchart1": {
										Version: "0.1.0",
									},
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "application",
				AllExecutors: map[string]executor.Executor{
					executor.HelmReleaseForward{}.GetName(): executor.HelmReleaseForward{},
				},
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1-application1": {
								Name:         "application1-application1",
								ChartName:    "application1",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{"subchart1":{"enabled":false},"subchart2":{"enabled":false},"subchart3":{"enabled":false}}`),
									},
								},
								Dependencies: []string{"application1-subchart1", "application1-subchart2", "application1-subchart3"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
							"application1-subchart1": {
								Name:         "application1-subchart1",
								ChartName:    utils.GetSubchartName("application1", "subchart1"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent:       "application1",
								Dependencies: []string{"application1-subchart2"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
							"application1-subchart2": {
								Name:         "application1-subchart2",
								ChartName:    utils.GetSubchartName("application1", "subchart2"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent: "application1",
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
							"application1-subchart3": {
								Name:         "application1-subchart3",
								ChartName:    utils.GetSubchartName("application1", "subchart3"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent:       "application1",
								Dependencies: []string{"application1-subchart1", "application1-subchart2"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
						},
					},
					"application2": {
						Name:         "application2",
						Dependencies: []string{"application1"},
						Tasks: map[string]*TaskNode{
							"application2-application2": {
								Name:         "application2-application2",
								ChartName:    "application2",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{"subchart1":{"enabled":false}}`),
									},
								},
								Dependencies: []string{"application2-subchart1"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
							"application2-subchart1": {
								Name:         "application2-subchart1",
								ChartName:    utils.GetSubchartName("application2", "subchart1"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent: "application2",
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseForward{},
									},
								},
							},
						},
					},
					"application3": {
						Name:         "application3",
						Dependencies: []string{"application2"},
						Tasks: map[string]*TaskNode{
							"application3-application3": {
								Name:         "application3-application3",
								ChartName:    "application3",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewForwardGraph(tt.args.appGroup)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("NewForwardGraph() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_NewReverseGraph(t *testing.T) {
	type args struct {
		appGroup *v1alpha1.ApplicationGroup
	}
	tests := []struct {
		name string
		args args
		want *Graph
	}{
		{
			name: "Reverse Basic Ordered Graph",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "application1",
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application1",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application2",
									Dependencies: []string{"application1"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application2",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application3",
									Dependencies: []string{"application2"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application3",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "application",
				AllExecutors: map[string]executor.Executor{
					executor.HelmReleaseReverse{}.GetName(): executor.HelmReleaseReverse{},
				},
				Nodes: map[string]*AppNode{
					"application1": {
						Name:         "application1",
						Dependencies: []string{"application2"},
						Tasks: map[string]*TaskNode{
							"application1-application1": {
								Name:         "application1-application1",
								ChartName:    "application1",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
						},
					},
					"application2": {
						Name:         "application2",
						Dependencies: []string{"application3"},
						Tasks: map[string]*TaskNode{
							"application2-application2": {
								Name:         "application2-application2",
								ChartName:    "application2",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
						},
					},
					"application3": {
						Name: "application3",
						Tasks: map[string]*TaskNode{
							"application3-application3": {
								Name:         "application3-application3",
								ChartName:    "application3",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Applications with Subcharts",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "application1",
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application1",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
									Subcharts: []v1alpha1.DAG{
										{
											Name:         "subchart1",
											Dependencies: []string{"subchart2"},
										},
										{
											Name: "subchart2",
										},
										{
											Name:         "subchart3",
											Dependencies: []string{"subchart1", "subchart2"},
										},
									},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application2",
									Dependencies: []string{"application1"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application2",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
									Subcharts: []v1alpha1.DAG{
										{
											Name: "subchart1",
										},
									},
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name:         "application3",
									Dependencies: []string{"application2"},
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										Name:    "application3",
										Version: "0.1.0",
									},
									Release: &v1alpha1.Release{},
								},
							},
						},
					},
					Status: v1alpha1.ApplicationGroupStatus{
						Applications: []v1alpha1.ApplicationStatus{
							{
								Subcharts: map[string]v1alpha1.ChartStatus{
									"subchart1": {
										Version: "0.1.0",
									},
									"subchart2": {
										Version: "0.1.0",
									},
									"subchart3": {
										Version: "0.1.0",
									},
								},
							},
							{
								Subcharts: map[string]v1alpha1.ChartStatus{
									"subchart1": {
										Version: "0.1.0",
									},
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "application",
				AllExecutors: map[string]executor.Executor{
					executor.HelmReleaseReverse{}.GetName(): executor.HelmReleaseReverse{},
				},
				Nodes: map[string]*AppNode{
					"application1": {
						Name:         "application1",
						Dependencies: []string{"application2"},
						Tasks: map[string]*TaskNode{
							"application1-application1": {
								Name:         "application1-application1",
								ChartName:    "application1",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{"subchart1":{"enabled":false},"subchart2":{"enabled":false},"subchart3":{"enabled":false}}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
							"application1-subchart1": {
								Name:         "application1-subchart1",
								ChartName:    utils.GetSubchartName("application1", "subchart1"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent:       "application1",
								Dependencies: []string{"application1-application1", "application1-subchart3"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
							"application1-subchart2": {
								Name:         "application1-subchart2",
								ChartName:    utils.GetSubchartName("application1", "subchart2"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Dependencies: []string{"application1-application1", "application1-subchart1", "application1-subchart3"},
								Parent:       "application1",
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
							"application1-subchart3": {
								Name:         "application1-subchart3",
								ChartName:    utils.GetSubchartName("application1", "subchart3"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent:       "application1",
								Dependencies: []string{"application1-application1"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
						},
					},
					"application2": {
						Name:         "application2",
						Dependencies: []string{"application3"},
						Tasks: map[string]*TaskNode{
							"application2-application2": {
								Name:         "application2-application2",
								ChartName:    "application2",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{"subchart1":{"enabled":false}}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
							"application2-subchart1": {
								Name:         "application2-subchart1",
								ChartName:    utils.GetSubchartName("application2", "subchart1"),
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Parent:       "application2",
								Dependencies: []string{"application2-application2"},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
									},
								},
							},
						},
					},
					"application3": {
						Name: "application3",
						Tasks: map[string]*TaskNode{
							"application3-application3": {
								Name:         "application3-application3",
								ChartName:    "application3",
								ChartVersion: "0.1.0",
								Release: &v1alpha1.Release{
									Values: &apiextensionsv1.JSON{
										Raw: []byte(`{}`),
									},
								},
								Executors: map[string]*ExecutorNode{
									"helmrelease": {
										Name:     "helmrelease",
										Executor: executor.HelmReleaseReverse{},
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
			got := NewReverseGraph(tt.args.appGroup)
			sortGraph(got)
			sortGraph(tt.want)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("NewReverseGraph() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_Diff(t *testing.T) {
	type args struct {
		a *Graph
		b *Graph
	}
	tests := []struct {
		name string
		args args
		want *Graph
	}{
		{
			name: "Basic Diff",
			args: args{
				a: &Graph{
					Name: "firstGraph",
					Nodes: map[string]*AppNode{
						"application1": {
							Name: "application1",
							Tasks: map[string]*TaskNode{
								"application1": {
									Name: "application1",
								},
							},
						},
					},
				},
				b: &Graph{
					Name: "secondGraph",
					Nodes: map[string]*AppNode{
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "firstGraph",
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1": {
								Name: "application1",
							},
						},
					},
				},
			},
		},
		{
			name: "Diff with Same Node",
			args: args{
				a: &Graph{
					Name: "firstGraph",
					Nodes: map[string]*AppNode{
						"application1": {
							Name: "application1",
							Tasks: map[string]*TaskNode{
								"application1": {
									Name: "application1",
								},
							},
						},
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
							},
						},
					},
				},
				b: &Graph{
					Name: "secondGraph",
					Nodes: map[string]*AppNode{
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "firstGraph",
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1": {
								Name: "application1",
							},
						},
					},
				},
			},
		},
		{
			name: "Different Subtasks in the First Graph",
			args: args{
				a: &Graph{
					Name: "firstGraph",
					Nodes: map[string]*AppNode{
						"application1": {
							Name: "application1",
							Tasks: map[string]*TaskNode{
								"application1": {
									Name: "application1",
								},
							},
						},
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
								"subtask1": {
									Name: "subtask1",
								},
								"subtask2": {
									Name: "subtask2",
								},
							},
						},
					},
				},
				b: &Graph{
					Name: "secondGraph",
					Nodes: map[string]*AppNode{
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "firstGraph",
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1": {
								Name: "application1",
							},
						},
					},
					"application2": {
						Name: "application2",
						Tasks: map[string]*TaskNode{
							"subtask1": {
								Name: "subtask1",
							},
							"subtask2": {
								Name: "subtask2",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diff(tt.args.a, tt.args.b)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("GetDiff() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_Combine(t *testing.T) {
	type args struct {
		a *Graph
		b *Graph
	}
	tests := []struct {
		name string
		args args
		want *Graph
	}{
		{
			name: "Basic Combine",
			args: args{
				a: &Graph{
					Name: "firstGraph",
					Nodes: map[string]*AppNode{
						"application1": {
							Name: "application1",
							Tasks: map[string]*TaskNode{
								"application1": {
									Name: "application1",
								},
							},
						},
					},
				},
				b: &Graph{
					Name: "secondGraph",
					Nodes: map[string]*AppNode{
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "firstGraph",
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1": {
								Name: "application1",
							},
						},
					},
					"application2": {
						Name: "application2",
						Tasks: map[string]*TaskNode{
							"application2": {
								Name: "application2",
							},
						},
					},
				},
			},
		},
		{
			name: "Combine Graphs that have Subtasks",
			args: args{
				a: &Graph{
					Name: "firstGraph",
					Nodes: map[string]*AppNode{
						"application1": {
							Name: "application1",
							Tasks: map[string]*TaskNode{
								"application1": {
									Name: "application1",
								},
							},
						},
						"application2": {
							Name: "application2",
							Tasks: map[string]*TaskNode{
								"application2": {
									Name: "application2",
								},
								"subtask1": {
									Name: "subtask1",
								},
								"subtask2": {
									Name: "subtask2",
								},
							},
						},
					},
				},
				b: &Graph{
					Name: "secondGraph",
					Nodes: map[string]*AppNode{
						"application3": {
							Name: "application3",
							Tasks: map[string]*TaskNode{
								"application3": {
									Name: "application3",
								},
								"subtask1": {
									Name: "subtask1",
								},
								"subtask2": {
									Name: "subtask2",
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name: "firstGraph",
				Nodes: map[string]*AppNode{
					"application1": {
						Name: "application1",
						Tasks: map[string]*TaskNode{
							"application1": {
								Name: "application1",
							},
						},
					},
					"application2": {
						Name: "application2",
						Tasks: map[string]*TaskNode{
							"application2": {
								Name: "application2",
							},
							"subtask1": {
								Name: "subtask1",
							},
							"subtask2": {
								Name: "subtask2",
							},
						},
					},
					"application3": {
						Name: "application3",
						Tasks: map[string]*TaskNode{
							"application3": {
								Name: "application3",
							},
							"subtask1": {
								Name: "subtask1",
							},
							"subtask2": {
								Name: "subtask2",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Combine(tt.args.a, tt.args.b)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("GetDiff() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func sortGraph(graph *Graph) {
	for _, elem := range graph.Nodes {
		sort.Strings(elem.Dependencies)
		for _, task := range elem.Tasks {
			sort.Strings(task.Dependencies)
		}
	}
}

func Test_clearDependencies(t *testing.T) {
	type args struct {
		graph *Graph
	}
	tests := []struct {
		name string
		args args
		want *Graph
	}{
		{
			name: "Clear Dependencies at Each Level",
			args: args{
				graph: &Graph{
					Name: "application",
					Nodes: map[string]*AppNode{
						"application1": {
							Dependencies: []string{"application2"},
							Tasks: map[string]*TaskNode{
								"application1": {
									Dependencies: []string{"application2", "application1"},
								},
							},
						},
						"application2": {
							Name:         "application2",
							Dependencies: []string{"application3"},
							Tasks: map[string]*TaskNode{
								"application2": {
									Dependencies: []string{},
								},
							},
						},
						"application3": {
							Name: "application3",
							Tasks: map[string]*TaskNode{
								"application3": {
									Dependencies: []string{"here", "there"},
								},
							},
						},
					},
				},
			},
			want: &Graph{
				Name:         "application",
				AllExecutors: map[string]executor.Executor{},
				Nodes: map[string]*AppNode{
					"application1": {
						Tasks: map[string]*TaskNode{
							"application1": {},
						},
					},
					"application2": {
						Name: "application2",
						Tasks: map[string]*TaskNode{
							"application2": {},
						},
					},
					"application3": {
						Name: "application3",
						Tasks: map[string]*TaskNode{
							"application3": {},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.graph.clear()
			if !cmp.Equal(got, tt.want) {
				t.Errorf("clearDependencies() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_subChartValues(t *testing.T) {
	type args struct {
		sc string
		av apiextensionsv1.JSON
	}
	tests := []struct {
		name string
		args args
		want apiextensionsv1.JSON
	}{
		{
			name: "withGlobalSuchart",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(`{"global":{"keyG":"valueG"},"subchart":{"keySC":"valueSC"}}`),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte(`{"global":{"keyG":"valueG"},"keySC":"valueSC"}`),
			},
		},
		{
			name: "withOnlyGlobal",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(`{"global":{"keyG":"valueG"}}`),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte(`{"global":{"keyG":"valueG"}}`),
			},
		},
		{
			name: "withOnlySubchart",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(`{"subchart":{"keySC":"valueSC"}}`),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte(`{"keySC":"valueSC"}`),
			},
		},
		{
			name: "withNone",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(""),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := make(map[string]interface{})
			_ = json.Unmarshal(tt.args.av.Raw, &values)
			if got, _ := SubChartValues(tt.args.sc, values); !cmp.Equal(*got, tt.want) {
				t.Errorf("subchartValues() = %v", cmp.Diff(*got, tt.want))
			}
		})
	}
}
