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
			want: GetHash("")[0:hashedAppNameMaxLen] + "-",
		},
		{
			name: "testing empty subchart name",
			args: args{
				appName: "10charhash",
				scName:  "",
			},
			want: GetHash("10charhash")[0:hashedAppNameMaxLen] + "-",
		},
		{
			name: "testing empty application name",
			args: args{
				appName: "",
				scName:  "myapp-name",
			},
			want: GetHash("")[0:hashedAppNameMaxLen] + "-myapp-name",
		},
		{
			name: "testing subchart name length == 52",
			args: args{
				appName: "appHash",
				scName:  "thisismychart-withbigname-equalto53chars000000000009",
			},
			want: GetHash("appHash")[0:hashedAppNameMaxLen] + "-thisismychart-withbigname-equalto53chars000000000009",
		},
		{
			name: "testing subchart name length > 52",
			args: args{
				appName: "appHash",
				scName:  "thisismychart-withbigname-greaterthan53chars0987654321abcde",
			},
			want: GetHash("appHash")[0:hashedAppNameMaxLen] + "-thisismychart-withbigname-greaterthan53chars09876543",
		},
		{
			name: "testing subchart name length > 63",
			args: args{
				appName: "appHash",
				scName:  "thisismyappchart-withbigname-greaterthan63chars0987654321abcde123456789",
			},
			want: GetHash("appHash")[0:hashedAppNameMaxLen] + "-thisismyappchart-withbigname-greaterthan63chars09876",
		},
		{
			name: "testing DNS1123 incompatible subchart name",
			args: args{
				appName: "appHash",
				scName:  "thisismyappchart_withbigname_greaterthan63chars0987654321abcde123456789",
			},
			want: GetHash("appHash")[0:hashedAppNameMaxLen] + "-thisismyappchart-withbigname-greaterthan63chars09876",
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
