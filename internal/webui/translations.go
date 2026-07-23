package webui

import (
	"encoding/json"
	"fmt"
	"strings"
)

const defaultLanguage = "en"

type catalog map[string]map[string]string

func loadCatalog() (catalog, error) {
	translations := make(catalog, 2)
	for _, language := range []string{"en", "it"} {
		content, err := files.ReadFile("translations/" + language + ".json")
		if err != nil {
			return nil, fmt.Errorf("read %s translations: %w", language, err)
		}
		messages := make(map[string]string)
		if err := json.Unmarshal(content, &messages); err != nil {
			return nil, fmt.Errorf("parse %s translations: %w", language, err)
		}
		translations[language] = messages
	}
	return translations, nil
}

func (c catalog) text(language, key string) string {
	language = normalizeLanguage(language)
	if message := c[language][key]; message != "" {
		return message
	}
	if message := c[defaultLanguage][key]; message != "" {
		return message
	}
	return key
}

func normalizeLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if separator := strings.IndexAny(language, "-_"); separator >= 0 {
		language = language[:separator]
	}
	if language == "it" {
		return "it"
	}
	return defaultLanguage
}
