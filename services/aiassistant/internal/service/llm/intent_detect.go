package llm

import "strings"

func ExtractProductKeyword(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"帮我", "",
		"请", "",
		"推荐", "",
		"一下", "",
		"商品", "",
		"产品", "",
		"我想买", "",
		"我要买", "",
		"有没有", "",
		"吗", "",
		"？", "",
		"?", "",
	)

	keyword := strings.TrimSpace(replacer.Replace(trimmed))
	if keyword == "" {
		return trimmed
	}
	return keyword
}
