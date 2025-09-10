package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"iPodSonic/lib"
	"os"
	"strconv"
	"sync"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var longest_song_title int64 = 0

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

func mp3ConvertNeeded(file string, newQuality uint8) bool {
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

func coverConvertNeeded(file string, newWidth uint16) bool {
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

func processCover(song *lib.Song, songConfig *SongConfig, dirs *lib.Directories, debrief *Debrief) bool {
	probeOutput, err := ffmpeg.Probe(song.OriginalSongFileName)
	var probe FFProbe
	json.Unmarshal([]byte(probeOutput), &probe)

	var coverStream *Stream
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			coverStream = &stream
			break
		}
	}
	if coverStream != nil {
		song.OriginalCoverFileName = fmt.Sprintf("%s/%s %s.%s", dirs.OrigCoverDir, song.Album, song.Title, coverStream.CodecName)
		song.ConvertedCoverFileName = fmt.Sprintf("%s/%s %s.%s", dirs.ConvertedCoverDir, song.Album, song.Title, coverStream.CodecName)
		coverConvertNeeded := coverConvertNeeded(song.ConvertedCoverFileName, uint16(songConfig.coverSize))
		if coverConvertNeeded {
			err = lib.ExtractCover(*song, uint(songConfig.coverSize))
			if err != nil {
				return false
			}
			debrief.CoverConverted = true
		}
		return true
	}
	return false
}

type Stream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     uint16 `json:"width"`
}

type SongConfig struct {
	mp3        bool
	flat       bool
	coverSize  uint
	mp3Quality uint
}
type Debrief struct {
	Downloaded     bool
	CoverConverted bool
	MP3Converted   bool
}

func processSong(song lib.Song, songConfig SongConfig, dirs lib.Directories, wg *sync.WaitGroup, sem chan struct{}, deb chan string) {
	var debrief Debrief
	var debriefString string
	defer wg.Done()
	defer func() { deb <- debriefString }()
	sem <- struct{}{}
	defer func() { <-sem }() // Release semaphore when done

	song.Title = lib.SanitizeFAT32Filename(song.Title)
	song.Album = lib.SanitizeFAT32Filename(song.Album)

	song.OriginalSongFileName = fmt.Sprintf("%s/%s %s.%s", dirs.OrigSongDir, song.Album, song.Title, song.Suffix)
	song.ConvertedSongFileName = fmt.Sprintf("%s/%s %s.%s", dirs.ConvertedSongDir, song.Album, song.Title, "mp3")

	song.ConvertedSongWithCoverFileName = fmt.Sprintf("%s/%s %s.%s", dirs.CombinedSongDir, song.Album, song.Title, "mp3")
	song.OriginalSongWithCoverFileName = fmt.Sprintf("%s/%s %s.%s", dirs.CombinedSongDir, song.Album, song.Title, song.Suffix)

	if !songConfig.flat {
		album, err := lib.GetAlbum(song.AlbumID)
		os.MkdirAll(fmt.Sprintf("%s/%s/%s", dirs.CombinedSongDir, album.Artist, song.Album), os.ModePerm)
		track, err := strconv.ParseUint(song.Track, 10, 64)
		if err != nil {
			track = 0
		}
		song.ConvertedSongWithCoverFileName = fmt.Sprintf("%s/%s/%s/%03d %s.%s", dirs.CombinedSongDir, album.Artist, song.Album, track, song.Title, "mp3")
		song.OriginalSongWithCoverFileName = fmt.Sprintf("%s/%s/%s/%03d %s.%s", dirs.CombinedSongDir, album.Artist, song.Album, track, song.Title, song.Suffix)
	}

	info, err := os.Stat(song.OriginalSongFileName)

	if err != nil || info.Size() != song.Size {
		err = lib.DownloadSong(song)
		debrief.Downloaded = true
		if err != nil {
			fmt.Println("aborting download", err)
			return
		}
	}

	if songConfig.mp3 && mp3ConvertNeeded(song.ConvertedSongFileName, uint8(songConfig.mp3Quality)) {
		debrief.MP3Converted = true
		lib.ConvertToMP3(song, songConfig.mp3Quality)
	}

	hasCover := processCover(&song, &songConfig, &dirs, &debrief)
	if hasCover {
		inputSong := song.OriginalSongFileName
		outputSong := song.OriginalSongWithCoverFileName
		if songConfig.mp3 {
			inputSong = song.ConvertedSongFileName
			outputSong = song.ConvertedSongWithCoverFileName
		}
		music := ffmpeg.Input(inputSong)
		cover := ffmpeg.Input(song.ConvertedCoverFileName)
		ffmpeg.Output([]*ffmpeg.Stream{music, cover}, outputSong, ffmpeg.KwArgs{
			"c":             "copy",
			"metadata:s:v":  `comment="Cover (front)"`,
			"id3v2_version": 3,
		}, ffmpeg.KwArgs{
			"disposition:v": "attached_pic", // Mark video as attached picture
			"metadata:s:v":  `title="Album cover"`,
		}).OverWriteOutput().Run()
	} else {
		if songConfig.mp3 {
			os.RemoveAll(song.ConvertedSongWithCoverFileName)
			err := os.Link(song.ConvertedSongFileName, song.ConvertedSongWithCoverFileName)
			if err != nil {
				fmt.Println("couldnt hard link file", song.ConvertedSongFileName)
			}
		} else {
			os.RemoveAll(song.OriginalSongWithCoverFileName)
			err := os.Link(song.ConvertedSongFileName, song.OriginalSongWithCoverFileName)
			if err != nil {
				fmt.Println("couldnt hard link file", song.ConvertedSongFileName)
			}
		}
	}
	dynamicPadding := "%-" + strconv.FormatInt(longest_song_title+2, 10) + "s"
	debriefString = fmt.Sprintf(dynamicPadding, song.Title)
	if !debrief.Downloaded && !debrief.MP3Converted && !debrief.CoverConverted {
		debriefString = fmt.Sprintf("%s nothing to do!", debriefString)
	}
	if debrief.Downloaded {
		debriefString = fmt.Sprintf("%s downloaded", debriefString)
	}
	if debrief.MP3Converted {
		debriefString = fmt.Sprintf("%s MP3-converted", debriefString)
	}
	if debrief.CoverConverted {
		debriefString = fmt.Sprintf("%s cover-converted", debriefString)
	}
}

func main() {
	concurrency := flag.Int("concurrency", 5, "set how many tasks run at the same time")
	coverSize := flag.Uint("coversize", 150, "set coverart size to <coversize>x<coversize>")
	dir := flag.String("dir", "./rocksonic_songs", "the folder where rocksonic can work with and save songs")
	flat := flag.Bool("flat", false, "don't create a folder structure, put everything into one folder")
	mp3 := flag.Bool("mp3", false, "compress everything to 320kbps mp3")
	mp3Quality := flag.Uint("quality", 2, "only has an effect when converting to mp3. sets the mp3 quality. 0=best but largest 9=worst but smallest")
	subsonicUrl := flag.String("url", "nourl", "the full url to the subsonic api like http://my.subsonic.com/rest")
	userNameFlag := flag.String("user", "nouser", "your username")
	passwordFlag := flag.String("pass", "nopass", "your password")
	playList := flag.String("playlist", "nolist", "the id of the playlist you want to download <optional>")

	flag.Parse()

	if *subsonicUrl == "nourl" {
		fmt.Println("no -url found")
		return
	}
	if *userNameFlag == "nouser" {
		fmt.Println("no -user found")
		return
	}
	if *passwordFlag == "nopass" {
		fmt.Println("no -pass found")
		return
	}
	lib.SetServer(*subsonicUrl, *userNameFlag, *passwordFlag)

	ffmpeg.LogCompiledCommand = false

	lib.InitMagick()
	defer lib.TerminateMagick()

	var err error

	fmt.Println("Welcome to RockSonic!")
	var songs []lib.Song
	var playlistName string

	if *playList != "nolist" {
		songs, playlistName, err = lib.GetPlaylist(*playList)
	} else {
		songs, err = lib.GetStarred()
		playlistName = "favs"
	}
	if err != nil {
		panic(err)
	}

	dirs, err := lib.MakeDirs(*dir, playlistName, *mp3)

	if err != nil {
		panic(err)
	}

	songConfig := SongConfig{
		mp3:        *mp3,
		flat:       *flat,
		coverSize:  *coverSize,
		mp3Quality: *mp3Quality,
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, *concurrency)
	deb := make(chan string)

	for _, song := range songs {

		if int64(len(song.Title)) > longest_song_title {
			longest_song_title = int64(len(song.Title))
		}
	}
	for _, song := range songs {

		wg.Add(1)
		go processSong(song, songConfig, dirs, &wg, sem, deb)
		// fmt.Printf("%s -> %s -> %3s. %s\n", song.Artist, song.Album, song.Track, song.Title)
	}
	for i := 0; i < len(songs); i++ {
		debStr := <-deb
		fmt.Printf("%6d/%d %s\n", i+1, len(songs), debStr)
	}
	wg.Wait()
	fmt.Println(dirs.ConvertedSongDir)
}
