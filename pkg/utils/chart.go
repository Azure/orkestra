package utils

func GetSubchartName(appName, scName string) string {
	scName = TruncateString(scName, subchartNameMaxLen)
	appName = TruncateString(GetHash(appName), DNS1123NameMaximumLength-len(scName)-1)
	return ConvertToDNS1123(appName + "-" + scName)
}
