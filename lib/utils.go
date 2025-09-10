package utils

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// Matches FAT32 forbidden characters: < > : " / \ | ? *
	illegalChars = regexp.MustCompile(`[<>:"/\\|?*]`)

	// Matches Windows reserved filenames (case-insensitive)
	reservedNames = regexp.MustCompile(`^(?i)(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])$`)
)

// SanitizeFAT32Filename replaces illegal characters, trims invalid endings,
// and ensures the result is FAT32-safe.
func SanitizeFAT32Filename(name string) string {
	if name == "" {
		return "unnamed"
	}

	// Replace forbidden characters with underscore
	name = illegalChars.ReplaceAllString(name, "_")

	// Replace control characters (ASCII < 32)
	var b strings.Builder
	for _, r := range name {
		if r < 32 {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	name = b.String()

	// Trim spaces and dots at the end (FAT32 doesn't allow trailing dots/spaces)
	name = strings.TrimRight(name, " .")

	// Prevent empty names after trimming
	if name == "" {
		name = "unnamed"
	}

	// Handle reserved filenames (e.g., CON, AUX)
	if reservedNames.MatchString(name) {
		name = "_" + name
	}

	// FAT32 max length = 255 UTF-8 bytes
	for len(name) > 255 {
		// Shorten one rune at a time to avoid cutting multibyte chars
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}

	return name
}
