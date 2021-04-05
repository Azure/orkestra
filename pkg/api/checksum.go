package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/mitchellh/hashstructure/v2"
)

const (
	AppGroupCsumKey = "application-group-spec"
)

var (
	ErrChecksumGenerateFailure      = errors.New("checksum generate failure")
	ErrChecksumAppGroupSpecMismatch = errors.New("application group spec checksum mismatch")
)

func Checksum(ag *v1alpha1.ApplicationGroup) (map[string]string, error) {
	var (
		// reconcile bool = false
		err  error
		csum = make(map[string]string)
	)

	h, err := hash(ag.Spec)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", err.Error(), ErrChecksumGenerateFailure)
	}

	csum[AppGroupCsumKey] = h

	if ag.Status.Checksums == nil {
		return csum, ErrChecksumAppGroupSpecMismatch
	}

	if csum[AppGroupCsumKey] != ag.Status.Checksums[AppGroupCsumKey] {
		return csum, ErrChecksumAppGroupSpecMismatch
	}

	return csum, nil
}

func hash(v interface{}) (string, error) {
	hash, err := hashstructure.Hash(v, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(hash, 10), nil
}
