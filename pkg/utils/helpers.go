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

// JoinForDNS1123 concatenates two given strings separated by "-", with resulting
// string <= 63 chars.
//
// Max len limit enforced by DNS1123 is 63 chars. The resulting string has
// at least 10 chars from a, 1 sep char (i.e. "-"), and remaining from b.
func JoinForDNS1123(a, b string) string {
	b = TruncateString(b, 52)
	a = TruncateString(a, 62-len(b))
	return a + "-" + b
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
