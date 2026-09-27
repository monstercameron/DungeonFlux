package i18n

// Message is a localizable view string carried as a key plus arguments.
// Count selects the plural form; Fallback renders when no catalog holds Key.
type Message struct {
	Key      string
	Args     map[string]string
	Count    int
	Fallback string
}

// Render resolves a message against a catalog for a locale.
func (m Message) Render(c Catalog, locale string) string {
	return c.T(locale, m.Key, m.Args, m.Count, m.Fallback)
}

// Text builds a simple non-plural message with a fallback.
func Text(key, fallback string) Message {
	return Message{Key: key, Fallback: fallback}
}

// WithArgs attaches formatting arguments to a message.
func (m Message) WithArgs(args map[string]string) Message {
	clone := m.Args
	if clone == nil && len(args) > 0 {
		clone = make(map[string]string, len(args))
	} else if len(args) > 0 {
		next := make(map[string]string, len(clone)+len(args))
		for key, value := range clone {
			next[key] = value
		}
		clone = next
	}
	for key, value := range args {
		clone[key] = value
	}
	m.Args = clone
	return m
}

// WithCount attaches a plural count to a message.
func (m Message) WithCount(count int) Message {
	m.Count = count
	return m
}
