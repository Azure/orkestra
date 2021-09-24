package utils

func GetSubchartName(appName, scName string) string {
	appName = TruncateString(GetHash(appName), hashedAppNameMaxLen)
	return ConvertToDNS1123(appName + "-" + scName)
}
