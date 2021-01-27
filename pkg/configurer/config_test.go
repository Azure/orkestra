package configurer

import (
	"errors"
	"github.com/Azure/Orkestra/pkg/registry"
	"reflect"
	"testing"
)

func TestController_RegistryConfig(t *testing.T) {
	type fields struct {
		Registries map[string]*registry.Config
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *registry.Config
		wantErr error
	}{
		{
			name: "exists",
			fields: fields{
				Registries: map[string]*registry.Config{
					"registry-1": {
						Hostname: StrToStrPtr("https://registry-1.acme.com:443"),
						Auth:     registry.NewBasicAuth("admin", "admin"),
					},
					"registry-2": {
						Hostname: StrToStrPtr("http://registry-2.acme.com"),
					},
				},
			},
			args: args{
				key: "registry-1",
			},
			want: &registry.Config{
				Hostname: StrToStrPtr("https://registry-1.acme.com:443"),
				Auth: &registry.BasicHTTPAuth{
					Username: "admin",
					Password: "admin",
				},
			},
			wantErr: nil,
		},
		{
			name: "stagingexists",
			fields: fields{
				Registries: map[string]*registry.Config{
					"registry-1": {
						Hostname: StrToStrPtr("https://registry-1.acme.com:443"),
						Auth:     registry.NewBasicAuth("admin", "admin"),
						Staging:  true,
					},
				},
			},
			args: args{
				key: "registry-1",
			},
			want: &registry.Config{
				Hostname: StrToStrPtr("https://registry-1.acme.com:443"),
				Auth: &registry.BasicHTTPAuth{
					Username: "admin",
					Password: "admin",
				},
				Staging: true,
			},
			wantErr: nil,
		},
		{
			name: "noexists",
			fields: fields{
				Registries: map[string]*registry.Config{
					"registry-1": {
						Hostname: StrToStrPtr("https://registry-1.acme.com:443"),
						Auth:     registry.NewBasicAuth("admin", "admin"),
					},
					"registry-2": {
						Hostname: StrToStrPtr("http://registry-2.acme.com"),
					},
				},
			},
			args: args{
				key: "registry-3",
			},
			want:    nil,
			wantErr: errRegistryNotFound,
		},
		{
			name: "empty",
			fields: fields{
				Registries: nil,
			},
			args: args{
				key: "registry-1",
			},
			want:    nil,
			wantErr: errEmptyRegistries,
		},
		{
			name: "empty",
			fields: fields{
				Registries: nil,
			},
			args: args{
				key: "",
			},
			want:    nil,
			wantErr: errEmptyKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				Registries: tt.fields.Registries,
			}
			got, err := c.RegistryConfig(tt.args.key)
			if err != nil {
				if tt.wantErr == nil {
					t.Errorf("RegistryConfig() error = %v, wantErr %v", err, tt.wantErr)
					return
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("RegistryConfig() error - error expected = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RegistryConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
