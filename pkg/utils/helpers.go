package utils

import (
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"gopkg.in/yaml.v2"
	"strings"
)

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

func ConvertSliceToDNS1123(in []string) []string {
	out := []string{}
	for _, s := range in {
		out = append(out, ConvertToDNS1123(s))
	}
	return out
}

func ToStrPtr(in string) *string {
	return &in
}

func HrToYaml(hr fluxhelmv2beta1.HelmRelease) string {
	b, err := yaml.Marshal(hr)
	if err != nil {
		return ""
	}

	return string(b)
}
