package configurer

import (
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
	"testing"
)

func TestNewConfigurer(t *testing.T) {
	type args struct {
		loc string
	}
	tests := []struct {
		name    string
		args    args
		want    *Configurer
		wantErr bool
	}{
		{
			name: "testwith",
			args: args{
				loc: "./testwith/config.yaml",
			},
			want: &Configurer{
				cfg: &viper.Viper{},
				Ctrl: &Controller{
					Registries: map[string]*registry.Config{
						"registry-1": {
							Hostname: strToStrPtr("https://registry-1.acme.com:443"),
							Auth:     registry.NewBasicAuth("admin", "admin"),
						},
						"registry-2": {
							Hostname: strToStrPtr("http://registry-2.acme.com"),
							Staging:  true,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfigurer(tt.args.loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfigurer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want.Ctrl, got.Ctrl); diff != "" {
				t.Errorf("NewConfigurer() mismatch (-want, +got)\n%s", diff)
			}
		})
	}
}

func strToStrPtr(s string) *string {
	return &s
}
