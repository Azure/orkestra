package registry

import (
	"github.com/go-logr/logr"
	test "github.com/go-logr/logr/testing"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"reflect"
	"testing"
)

var logger = &test.NullLogger{}

func TestLoad(t *testing.T) {
	type args struct {
		chartLocation string
		cleanup       bool
		logr          logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    *chart.Chart
		wantErr bool
	}{
		{
			name: "loads-packaged",
			args: args{
				chartLocation: "testwith/argo-0.1.0.tgz",
				cleanup:       false,
				logr:          logger,
			},
			want:    loadTestPackagedChart("testwith/argo-0.1.0.tgz"),
			wantErr: false,
		},
		{
			name: "errors",
			args: args{
				chartLocation: "testwith/",
				cleanup:       true,
				logr:          logger,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args.chartLocation, tt.args.cleanup, tt.args.logr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Todo(kushthedude) : needs some more detailed info in the helmReleaseSpec
//func TestFetch1(t *testing.T) {
//	type args struct {
//		helmReleaseSpec v1.HelmReleaseSpec
//		location        string
//		logr            logr.Logger
//		cfg             *Config
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    string
//		wantErr bool
//	}{
//		{
//			name: "works",
//			args: {
//				helmReleaseSpec: &v1.HelmReleaseSpec{
//					ChartSource: v1.ChartSource{
//						RepoChartSource:,
//					},
//				},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := Fetch(tt.args.helmReleaseSpec, tt.args.location, tt.args.logr, tt.args.cfg)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if got != tt.want {
//				t.Errorf("Fetch() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
func loadTestPackagedChart(chartPath string) *chart.Chart {
	ch, err := loader.LoadFile(chartPath)
	if err != nil {
		return nil
	}
	return ch
}
