package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"sigs.k8s.io/yaml"
)

func ConvertToDNS1123(in string) string {
	return strings.ReplaceAll(in, "_", "-")
}

func ConvertSliceToDNS1123(in []string) []string {
	out := []string{}
	for _, s := range in {
		out = append(out, ConvertToDNS1123(s))
	}
	return out
}

func GetHash(in string) string {
	h := sha256.New()
	h.Write([]byte(in))
	return hex.EncodeToString(h.Sum(nil))
}

func TruncateString(in string, num int) string {
	out := in
	if len(in) > num {
		out = in[0:num]
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
