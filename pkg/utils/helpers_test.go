package utils

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	type args struct {
		in  string
		num int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testing empty",
			args: args{
				in:  "",
				num: 0,
			},
			want: "",
		},
		{
			name: "testing empty with > 0 requested length",
			args: args{
				in:  "",
				num: 5,
			},
			want: "",
		},
		{
			name: "testing input string length == requested length",
			args: args{
				in:  "hello, don't truncate me",
				num: 24,
			},
			want: "hello, don't truncate me",
		},
		{
			name: "testing input string length < requested length",
			args: args{
				in:  "hello again, don't truncate me",
				num: 63,
			},
			want: "hello again, don't truncate me",
		},
		{
			name: "testing input string length > requested length",
			args: args{
				in:  "truncate_this_string_so_that_its_length_is_less_than_sixty_three_characters",
				num: 63,
			},
			want: "truncate_this_string_so_that_its_length_is_less_than_sixty_thre",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateString(tt.args.in, tt.args.num); got != tt.want {
				t.Errorf("TruncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinForDNS1123(t *testing.T) {
	type args struct {
		a string
		b string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testing empty",
			args: args{
				a: "",
				b: "",
			},
			want: "-",
		},
		{
			name: "testing empty b",
			args: args{
				a: "10charhash",
				b: "",
			},
			want: "10charhash-",
		},
		{
			name: "testing empty a",
			args: args{
				a: "",
				b: "myapp-name",
			},
			want: "-myapp-name",
		},
		{
			name: "testing app name length < 52",
			args: args{
				a: "961b6dd3ede3cb8ecbaacbd68de040cd78eb2ed5889130cceb4c49268ea4d506",
				b: "myapp",
			},
			want: "961b6dd3ede3cb8ecbaacbd68de040cd78eb2ed5889130cceb4c49268-myapp",
		},
		{
			name: "testing app name length == 52",
			args: args{
				a: "961b6dd3ede3cb8ecbaacbd68de040cd78eb2ed5889130cceb4c49268ea4d506",
				b: "thisismyapp-withbigname-equalto53chars00000000000000",
			},
			want: "961b6dd3ed-thisismyapp-withbigname-equalto53chars00000000000000",
		},
		{
			name: "testing app name length > 52",
			args: args{
				a: "961b6dd3ede3cb8ecbaacbd68de040cd78eb2ed5889130cceb4c49268ea4d506",
				b: "thisismyapp-withbigname-greaterthan53chars0987654321abcde",
			},
			want: "961b6dd3ed-thisismyapp-withbigname-greaterthan53chars0987654321",
		},
		{
			name: "testing app name length > 63",
			args: args{
				a: "961b6dd3ede3cb8ecbaacbd68de040cd78eb2ed5889130cceb4c49268ea4d506",
				b: "thisismyapp-withbigname-greaterthan53chars0987654321abcde123456789",
			},
			want: "961b6dd3ed-thisismyapp-withbigname-greaterthan53chars0987654321",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinForDNS1123(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("TestJoinForDNS1123() = %v, want %v", got, tt.want)
			}
		})
	}
}
