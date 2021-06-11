package utils

import (
	"testing"
)

func TestConvertToDNS1123(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testing valid name",
			args: args{
				in: "good-small-name",
			},
			want: "good-small-name",
		},
		{
			name: "testing invalid name",
			args: args{
				in: "tOk3_??ofTHE-Runner",
			},
			want: "tok3---ofthe-runner",
		},
		{
			name: "testing all characters are invalid",
			args: args{
				in: "?.?&^%#$@_??",
			},
			want: GetHash("?.?&^%#$@_??")[0:63],
		},
		{
			name: "testing invalid start chars",
			args: args{
				in: "----??tOk3_??ofTHE-Runner",
			},
			want: "tok3---ofthe-runner",
		},
		{
			name: "testing very long name",
			args: args{
				in: "very-long-name------------------------------------------------end",
			},
			want: "very-long-name------------------------------------------------e",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToDNS1123(tt.args.in); got != tt.want {
				t.Errorf("ConvertDNS1123() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
