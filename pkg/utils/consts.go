package utils

const (
	DNS1123NameMaximumLength    = 63
	DNS1123NotAllowedChars      = "[^-a-z0-9]"
	DNS1123NotAllowedStartChars = "^[^a-z0-9]+"

	// hashedAppNameMaxLen is the maximum length of application name hash that is
	hashedAppNameMaxLen = 10
)
