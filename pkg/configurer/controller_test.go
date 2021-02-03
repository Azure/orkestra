package configurer

import (
	"reflect"
	"testing"

	"github.com/Azure/Orkestra/pkg/registry"
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
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				Registries: tt.fields.Registries,
			}
			got, err := c.RegistryConfig(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegistryConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RegistryConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
