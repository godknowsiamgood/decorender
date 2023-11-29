package resources

import (
	"strings"
)

func IsLocalResource(fileName string) bool {
	return !strings.HasPrefix(fileName, "http://") && !strings.HasPrefix(fileName, "https://")
}
