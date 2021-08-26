package workflow

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"testing"
)

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
			if got, _ := subChartValues(tt.args.sc, values); !cmp.Equal(*got, tt.want) {
				t.Errorf("subchartValues() = %v", cmp.Diff(*got, tt.want))
			}
		})
	}
}
