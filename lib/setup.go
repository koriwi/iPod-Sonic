package lib

import (
	"fmt"
	"os"
)

type Directories struct {
	OrigDir           string
	ConvertedDir      string
	ConvertedSongDir  string
	CombinedSongDir   string
	OrigSongDir       string
	OrigCoverDir      string
	ConvertedCoverDir string
}

func MakeDirs(rootDir string, playlistName string, mp3 bool) (Directories, error) {

	err := os.MkdirAll(rootDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}
	origDir := fmt.Sprintf("%s/.original", rootDir)
	err = os.MkdirAll(origDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}

	convertedDir := fmt.Sprintf("%s/.converted", rootDir)
	err = os.MkdirAll(convertedDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}

	convertedSongDir := fmt.Sprintf("%s/songs", convertedDir)
	err = os.MkdirAll(convertedSongDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}
	combinedSongDir := fmt.Sprintf("%s/", rootDir)
	sanitized := SanitizeFAT32Filename(playlistName)
	combinedSongDir = fmt.Sprintf("%s/%s", combinedSongDir, sanitized)
	if mp3 {
		combinedSongDir = fmt.Sprintf("%s_mp3", combinedSongDir)
	}

	err = os.MkdirAll(combinedSongDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}

	origSongDir := fmt.Sprintf("%s/songs", origDir)
	err = os.MkdirAll(origSongDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}

	origCoverDir := fmt.Sprintf("%s/covers", origDir)
	err = os.MkdirAll(origCoverDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}

	convertedCoverDir := fmt.Sprintf("%s/covers", convertedDir)
	err = os.MkdirAll(convertedCoverDir, os.ModePerm)
	if err != nil {
		return Directories{}, err
	}
	return Directories{
		OrigDir:           origDir,
		OrigCoverDir:      origCoverDir,
		OrigSongDir:       origSongDir,
		ConvertedDir:      convertedDir,
		ConvertedSongDir:  convertedSongDir,
		ConvertedCoverDir: convertedCoverDir,
		CombinedSongDir:   combinedSongDir,
	}, nil
}
