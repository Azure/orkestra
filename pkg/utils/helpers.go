package utils

import "strings"

func ConvertToDNS1123(in string) string {
	return strings.ReplaceAll(in, "_", "-")
}

func ToInitials(in string) (out string) {
	in = ConvertToDNS1123(in)
	parts := strings.Split(in, "-")

	for _, part := range parts {
		out += string(part[0])
	}
	return out
}
