package workflow

import (
	"os"
)

func GetNamespace() string {
	if ns, ok := os.LookupEnv("WORKFLOW_NAMESPACE"); ok {
		return ns
	}
	return "orkestra"
}
