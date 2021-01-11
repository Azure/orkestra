package registry

import (
	v1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	"github.com/go-logr/logr"
	"testing"
)

func TestFetch(t *testing.T) {
	type args struct {
		helmReleaseSpec v1.HelmReleaseSpec
		location        string
		logr            logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Fetch(tt.args.helmReleaseSpec, tt.args.location, tt.args.logr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fetch() got = %v, want %v", got, tt.want)
			}
		})
	}
}
