// Package i18n provides simple internationalisation for server messages.
// Messages are loaded from a TOML file with sections per language.
// Per-player language preference falls back to the server default ("en").
package i18n

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
)

// I18n holds all loaded translations. Safe for concurrent use.
type I18n struct {
	mu       sync.RWMutex
	default_ string
	msgs     map[string]map[string]string
}

// New creates an empty I18n with the given default language.
func New(defaultLang string) *I18n {
	if defaultLang == "" {
		defaultLang = "en"
	}
	return &I18n{
		default_: defaultLang,
		msgs:     make(map[string]map[string]string),
	}
}

// Load reads translations from a TOML file. Each [lang] section is a
// flat key = "message" table. Returns nil if the file doesn't exist.
func (i *I18n) Load(path string) error {
	var raw map[string]map[string]string
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return fmt.Errorf("decode language file %s: %w", path, err)
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for lang, msgs := range raw {
		if i.msgs[lang] == nil {
			i.msgs[lang] = make(map[string]string, len(msgs))
		}
		for k, v := range msgs {
			i.msgs[lang][k] = v
		}
	}
	return nil
}

// Get returns the message for key in lang, falling back to the default
// language, then to the key itself if not found.
// Args are substituted via fmt.Sprintf if provided.
func (i *I18n) Get(lang, key string, args ...any) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	msg := i.lookup(lang, key)
	if msg == "" {
		msg = i.lookup(i.default_, key)
	}
	if msg == "" {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

func (i *I18n) lookup(lang, key string) string {
	if msgs, ok := i.msgs[lang]; ok {
		if m, ok := msgs[key]; ok {
			return m
		}
	}
	return ""
}

// Languages returns all loaded language codes.
func (i *I18n) Languages() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]string, 0, len(i.msgs))
	for lang := range i.msgs {
		out = append(out, lang)
	}
	return out
}
