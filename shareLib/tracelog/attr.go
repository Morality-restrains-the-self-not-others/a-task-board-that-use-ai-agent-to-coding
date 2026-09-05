package tracelog

// Attr is a key/value span attribute without importing OTEL in default builds.
type Attr struct {
	Key   string
	Value string
}

// String returns a string-valued span attribute.
func String(key, value string) Attr {
	return Attr{Key: key, Value: value}
}
