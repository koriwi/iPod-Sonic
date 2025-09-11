package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Stream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     uint16 `json:"width"`
}
type FFProbe struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}
type Format struct {
	Tags Tags `json:"tags"`
}
type Tags struct {
	RocksonicQuality string `json:"rocksonic_quality"`
}

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

func HasCover(fileName string) (*Stream, error) {

	probeOutput, err := ffmpeg.Probe(fileName)
	if err != nil {
		return nil, err
	}
	var probe FFProbe
	json.Unmarshal([]byte(probeOutput), &probe)

	var coverStream *Stream
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			coverStream = &stream
			break
		}
	}
	return coverStream, nil
}

func CoverConvertNeeded(file string, newWidth uint16) bool {
	_, err := os.Stat(file)
	if err != nil {
		return true
	}

	probeOutput, err := ffmpeg.Probe(file, []ffmpeg.KwArgs{{"select_streams": "v:0"}, {"of": "json"}}...)
	var probe FFProbe
	json.Unmarshal([]byte(probeOutput), &probe)
	if err != nil {
		return true
	}

	return probe.Streams[0].Width != newWidth
}

func MP3ConvertNeeded(file string, newQuality uint8) bool {
	_, err := os.Stat(file)
	if err != nil {
		return true
	}

	probeOutput, err := ffmpeg.Probe(file, []ffmpeg.KwArgs{{"show_format": ""}, {"print_format": "json"}}...)
	var probe FFProbe
	json.Unmarshal([]byte(probeOutput), &probe)
	if err != nil {
		return true
	}
	// fmt.Printf("%-100s %s %d\n", file, probe.Format.Tags.RocksonicQuality, newQuality)
	return probe.Format.Tags.RocksonicQuality != fmt.Sprintf("%d", newQuality)
}
