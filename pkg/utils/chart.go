package utils

// subchartNameMaxLen is the maximum length of a subchart name.
//
// The max name length limit enforced by DNS1123 is 63 chars. We reserve 10 chars
// for concatenating application name hash.
const subchartNameMaxLen = 53

func GetSubchartName(appName, scName string) string {
	scName = TruncateString(scName, subchartNameMaxLen)
	appName = TruncateString(GetHash(appName), DNS1123NameMaximumLength-len(scName)-1)
	return ConvertToDNS1123(appName + "-" + scName)
}
