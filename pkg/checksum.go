package pkg

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/mitchellh/hashstructure/v2"
)

const (
	appGroupCsumKey                = "checksum/application-group-spec"
	applicationCsumKeyPrefix       = "checksum/application-spec-"
	applicationValuesCsumKeyPrefix = "checksum/application-values-"
)

var (
	ErrChecksumGenerateFailure      = errors.New("checksum generate failure")
	ErrChecksumAppGroupSpecMismatch = errors.New("application group spec checksum mismatch")
	ErrChecksumAppSpecMismatch      = errors.New("application spec checksum mismatch")
	ErrChecksumAppValuesMismatch    = errors.New("application values checksum mismatch")
)

func Checksum(ag *v1alpha1.ApplicationGroup) (bool, map[string]string, error) {
	var (
		// reconcile bool = false
		err  error
		csum = make(map[string]string)
	)

	h, err := hash(ag.Spec)
	if err != nil {
		return false, nil, fmt.Errorf("%s : %w", err.Error(), ErrChecksumGenerateFailure)
	}

	csum[appGroupCsumKey] = h

	for _, application := range ag.Spec.Applications {
		applicationHash, err2 := hash(application.Spec)
		if err2 != nil {
			return false, nil, ErrChecksumGenerateFailure
		}

		valuesHash, err2 := hash(application.Spec.Overlays)
		if err2 != nil {
			return false, nil, ErrChecksumGenerateFailure
		}

		csum[applicationCsumKeyPrefix+application.Name] = applicationHash
		csum[applicationValuesCsumKeyPrefix+application.Name] = valuesHash
	}

	if ag.Status.Checksums != nil {
		for k, v := range csum {
			if strings.Contains(k, applicationCsumKeyPrefix) {
				if v != ag.Status.Checksums[k] {
					return true, csum, ErrChecksumAppSpecMismatch
				}
			}

			if strings.Contains(k, applicationValuesCsumKeyPrefix) {
				if v != ag.Status.Checksums[k] {
					return true, csum, ErrChecksumAppValuesMismatch
				}
			}
		}
		if csum[appGroupCsumKey] != ag.Status.Checksums[appGroupCsumKey] {
			return true, csum, ErrChecksumAppGroupSpecMismatch
		}
	}

	return false, csum, nil
}

func hash(v interface{}) (string, error) {
	hash, err := hashstructure.Hash(v, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(hash, 10), nil
}
