package workflow

import (
	"testing"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestBuild(t *testing.T) {
	type args struct {
		entry string
		nodes map[string]v1alpha13.NodeStatus
	}
	testmap := make(map[string]v1alpha13.NodeStatus)
	testmap["dummy"] = v1alpha13.NodeStatus{}

	tests := []struct {
		name string
		args args
		want *Graph
		err  error
	}{
		//
		// TODO: Add more test cases.
		//
		{
			name: "testing nil nodes",
			args: args{
				entry: "",
				nodes: nil,
			},
			want: nil,
			err:  ErrNoNodesFound,
		},
		{
			name: "testing unknown entry",
			args: args{
				entry: "unknown",
				nodes: testmap,
			},
			want: nil,
			err:  ErrEntryNodeNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(tt.args.entry, tt.args.nodes)
			if err != tt.err {
				t.Errorf("Build() error = %v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("Build() = %v, want %v", got, tt.want)
			}
		})
	}
}
