package common

import "strings"

func IsImageFormat(url string) bool {
	return strings.HasSuffix(url, ".jpg") ||
		strings.HasSuffix(url, ".jpeg") ||
		strings.HasSuffix(url, ".png") ||
		strings.HasSuffix(url, ".gif")
}
