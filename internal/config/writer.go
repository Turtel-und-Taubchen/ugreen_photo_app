package config

import "fmt"

// Writer holds the settings of the metadata writer.
type Writer struct {
	PreserveMtime bool
}

// WriterFromEnv reads PC_PRESERVE_MTIME (default true, only "true" or "false" are accepted).
func WriterFromEnv(get func(string) string) (Writer, error) {
	switch v := get("PC_PRESERVE_MTIME"); v {
	case "", "true":
		return Writer{PreserveMtime: true}, nil
	case "false":
		return Writer{PreserveMtime: false}, nil
	default:
		return Writer{}, fmt.Errorf("PC_PRESERVE_MTIME=%q is not valid (use true or false)", v)
	}
}
