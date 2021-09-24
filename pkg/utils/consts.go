package utils

import (
	"regexp"
)

const (
	DNS1123NameMaximumLength = 63

	// hashedAppNameMaxLen is the maximum length of application name hash that is
	hashedAppNameMaxLen = 10
)

var (
	dns1123NotAllowedCharsRegexp      = regexp.MustCompile("[^-a-z0-9]")  // treat as const
	dns1123NotAllowedStartCharsRegexp = regexp.MustCompile("^[^a-z0-9]+") // treat as const
)
