package common

func RemoveSingleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "'Hello'"
	if len(articleName) > 2 && articleName[0] == '\'' && articleName[len(articleName)-1] == '\'' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}

func RemoveDoubleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "\"Hello\""
	if len(articleName) > 2 && articleName[0] == '"' && articleName[len(articleName)-1] == '"' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}
