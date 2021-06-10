package utils

const (
	DNS1123NameMaximumLength    = 63
	DNS1123NotAllowedChars      = "[^-a-z0-9]"
	DNS1123NotAllowedStartChars = "^[^a-z0-9]+"

	// subchartNameMaxLen is the maximum length of a subchart name.
	//
	// The max name length limit enforced by DNS1123 is 63 chars. We reserve 10 chars
	// for concatenating application name hash.
	subchartNameMaxLen = 53
)
