package utils

import (
	"testing"
)

func TestGetSubchartName(t *testing.T) {
	type args struct {
		appName string
		scName  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testing empty",
			args: args{
				appName: "",
				scName:  "",
			},
			want: GetHash("")[0:62] + "-",
		},
		{
			name: "testing empty subchart name",
			args: args{
				appName: "10charhash",
				scName:  "",
			},
			want: GetHash("10charhash")[0:62] + "-",
		},
		{
			name: "testing empty application name",
			args: args{
				appName: "",
				scName:  "myapp-name",
			},
			want: GetHash("")[0:52] + "-myapp-name",
		},
		{
			name: "testing subchart name length < 53",
			args: args{
				appName: "appHash",
				scName:  "mychart",
			},
			want: GetHash("appHash")[0:55] + "-mychart",
		},
		{
			name: "testing subchart name length == 53",
			args: args{
				appName: "appHash",
				scName:  "thisismychart-withbigname-equalto53chars0000000000000",
			},
			want: GetHash("appHash")[0:9] + "-thisismychart-withbigname-equalto53chars0000000000000",
		},
		{
			name: "testing subchart name length > 53",
			args: args{
				appName: "appHash",
				scName:  "thisismychart-withbigname-greaterthan53chars0987654321abcde",
			},
			want: GetHash("appHash")[0:9] + "-thisismychart-withbigname-greaterthan53chars098765432",
		},
		{
			name: "testing subchart name length > 63",
			args: args{
				appName: "appHash",
				scName:  "thisismyappchart-withbigname-greaterthan63chars0987654321abcde123456789",
			},
			want: GetHash("appHash")[0:9] + "-thisismyappchart-withbigname-greaterthan63chars098765",
		},
		{
			name: "testing DNS1123 incompatible subchart name",
			args: args{
				appName: "appHash",
				scName:  "thisismyappchart_withbigname_greaterthan63chars0987654321abcde123456789",
			},
			want: GetHash("appHash")[0:9] + "-thisismyappchart-withbigname-greaterthan63chars098765",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSubchartName(tt.args.appName, tt.args.scName); got != tt.want {
				t.Errorf("TestGetSubchartReleaseName() = %v, want %v", got, tt.want)
			}
		})
	}
}
