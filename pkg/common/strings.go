package common

// IsStringInSlice returns true if string `str` is found in `slice`.
func IsStringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if str == s {
			return true
		}
	}
	return false
}
