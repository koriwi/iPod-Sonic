package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	utils "iPodSonic/sync/lib"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	magick "gopkg.in/gographics/imagick.v3/imagick"
)

var longest_song_title int64 = 0
var serverURL = ""
var userName = ""
var userPassword = ""

func getUrl(endpoint string, extraParams ...string) string {
	extraString := ""
	for _, param := range extraParams {
		extraString = fmt.Sprintf("%s&%s", extraString, param)
	}
	url := fmt.Sprintf("%s/%s?v=1.16.1&c=rocksonic&u=%s&p=%s%s", serverURL, endpoint, userName, userPassword, extraString)
	// fmt.Printf("%s/%s?v=1.16.1&c=rocksonic&u=%s&p=%s%s\n", serverURL, endpoint, userName, "*****", extraString)
	return url
}

type Song struct {
	ID                             string `xml:"id,attr"`
	Artist                         string `xml:"artist,attr"`
	Track                          string `xml:"track,attr"`
	Title                          string `xml:"title,attr"`
	Album                          string `xml:"album,attr"`
	AlbumID                        string `xml:"albumId,attr"`
	Suffix                         string `xml:"suffix,attr"`
	Size                           int64  `xml:"size,attr"`
	OriginalSongFileName           string
	OriginalCoverFileName          string
	OriginalSongWithCoverFileName  string
	ConvertedSongFileName          string
	ConvertedCoverFileName         string
	ConvertedSongWithCoverFileName string
}
type Starred struct {
	XMLName xml.Name `xml:"starred"`
	Songs   []Song   `xml:"song"`
}
type Playlist struct {
	XMLName xml.Name `xml:"playlist"`
	Name    string   `xml:"name,attr"`
	Songs   []Song   `xml:"entry"`
}
type Album struct {
	XMLName xml.Name `xml:"album"`
	Artist  string   `xml:"artist,attr"`
}
type SSResponse struct {
	XMLName  xml.Name `xml:"subsonic-response"`
	Starred  Starred
	Playlist Playlist
	Album    Album
}

func convertToMP3(song Song, quality uint) error {
	err := ffmpeg.Input(song.OriginalSongFileName).Output(
		song.ConvertedSongFileName,
		ffmpeg.KwArgs{"vn": "", "q:a": quality, "map_metadata": 0, "id3v2_version": 3, "write_id3v1": 1, "write_id3v2": 1, "metadata": fmt.Sprintf("rocksonic_quality=%d", quality), "y": ""},
	).OverWriteOutput().Run()
	if err != nil {
		return errors.New(fmt.Sprint("error extracting cover from", song.OriginalSongFileName, err))
	}

	return nil
}

func extractCover(song Song, coverStream Stream, coverSize uint) error {

	err := ffmpeg.Input(song.OriginalSongFileName).Output(
		song.OriginalCoverFileName,
		ffmpeg.KwArgs{"an": "", "update": 1, "pix_fmt": "yuvj420p", "color_range": "full", "colorspace": "bt470bg"},
	).OverWriteOutput().Run()
	if err != nil {
		return errors.New(fmt.Sprint("error extracting cover from", song.OriginalSongFileName, err))
	}

	mw := magick.NewMagickWand()
	defer mw.Destroy()
	err = mw.ReadImage(song.OriginalCoverFileName)
	if err != nil {
		return errors.New(fmt.Sprint("error reading extracted cover file", song.OriginalCoverFileName, err))
	}

	aspectRatio := float32(mw.GetImageWidth()) / float32(mw.GetImageHeight())
	var width = coverSize
	var height = uint(float32(coverSize) * aspectRatio)

	err = mw.ResizeImage(width, height, magick.FILTER_LANCZOS)
	if err != nil {
		return errors.New(fmt.Sprint("error resizing cover file", song.OriginalCoverFileName, err))
	}

	err = mw.SetImageCompressionQuality(75)
	if err != nil {
		return errors.New(fmt.Sprint("error compressing cover file", song.OriginalCoverFileName, err))
	}

	err = mw.StripImage()
	if err != nil {
		return errors.New(fmt.Sprint("error stripping cover file", song.OriginalCoverFileName, err))
	}

	err = mw.SetInterlaceScheme(magick.INTERLACE_NO)
	if err != nil {
		return errors.New(fmt.Sprint("error disabling interlace for cover file", song.OriginalCoverFileName, err))
	}

	err = mw.WriteImage(song.ConvertedCoverFileName)
	if err != nil {
		return errors.New(fmt.Sprint("error saving cover file", song.ConvertedCoverFileName, err))
	}

	return nil
}

func downloadSong(song Song) error {

	url := getUrl("download", "id="+string(song.ID))
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("could not download", song.Title)
		return errors.New("could not download " + song.Title + "\n" + err.Error())
	}
	defer resp.Body.Close()

	out, err := os.Create(song.OriginalSongFileName)
	if err != nil {
		// fmt.Println("file already exists", song.Title)
		return errors.New("could not create file " + song.Title + "\n" + err.Error())
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("could not save to file", song.Title)
		return errors.New("could not save to file " + song.Title + "\n" + err.Error())
	}
	return nil
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

func processCover(song *Song, songConfig *SongConfig, debrief *Debrief) bool {
	// move this to a function later
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
		song.OriginalCoverFileName = fmt.Sprintf("%s/%s %s.%s", songConfig.origCoverDir, song.Album, song.Title, coverStream.CodecName)
		song.ConvertedCoverFileName = fmt.Sprintf("%s/%s %s.%s", songConfig.convertedCoverDir, song.Album, song.Title, coverStream.CodecName)
		coverConvertNeeded := coverConvertNeeded(song.ConvertedCoverFileName, uint16(songConfig.coverSize))
		if coverConvertNeeded {
			err = extractCover(*song, *coverStream, uint(songConfig.coverSize))
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
	mp3               bool
	flat              bool
	coverSize         uint
	convertedDir      string
	convertedSongDir  string
	combinedSongDir   string
	origCoverDir      string
	origSongDir       string
	convertedCoverDir string
	mp3Quality        uint
}
type Debrief struct {
	Downloaded     bool
	CoverConverted bool
	MP3Converted   bool
}

func processSong(song Song, songConfig SongConfig, wg *sync.WaitGroup, sem chan struct{}, deb chan string) {
	var debrief Debrief
	var debriefString string
	defer wg.Done()
	defer func() { deb <- debriefString }()
	sem <- struct{}{}
	defer func() { <-sem }() // Release semaphore when done

	song.Title = utils.SanitizeFAT32Filename(song.Title)
	song.Album = utils.SanitizeFAT32Filename(song.Album)

	song.OriginalSongFileName = fmt.Sprintf("%s/%s %s.%s", songConfig.origSongDir, song.Album, song.Title, song.Suffix)
	song.ConvertedSongFileName = fmt.Sprintf("%s/%s %s.%s", songConfig.convertedSongDir, song.Album, song.Title, "mp3")

	song.ConvertedSongWithCoverFileName = fmt.Sprintf("%s/%s %s.%s", songConfig.combinedSongDir, song.Album, song.Title, "mp3")
	song.OriginalSongWithCoverFileName = fmt.Sprintf("%s/%s %s.%s", songConfig.combinedSongDir, song.Album, song.Title, song.Suffix)
	if !songConfig.flat {
		var resp *http.Response
		resp, err := http.Get(getUrl("getAlbum", fmt.Sprintf("id=%s", song.AlbumID)))
		var albumResult SSResponse

		err = xml.NewDecoder(resp.Body).Decode(&albumResult)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		os.MkdirAll(fmt.Sprintf("%s/%s/%s", songConfig.combinedSongDir, albumResult.Album.Artist, song.Album), os.ModePerm)
		track, err := strconv.ParseUint(song.Track, 10, 64)
		if err != nil {
			track = 0
		}
		song.ConvertedSongWithCoverFileName = fmt.Sprintf("%s/%s/%s/%03d %s.%s", songConfig.combinedSongDir, albumResult.Album.Artist, song.Album, track, song.Title, "mp3")
		song.OriginalSongWithCoverFileName = fmt.Sprintf("%s/%s/%s/%03d %s.%s", songConfig.combinedSongDir, albumResult.Album.Artist, song.Album, track, song.Title, song.Suffix)
	}

	info, err := os.Stat(song.OriginalSongFileName)

	if err != nil || info.Size() != song.Size {
		err = downloadSong(song)
		debrief.Downloaded = true
		if err != nil {
			fmt.Println("aborting download", err)
			return
		}
	}

	if songConfig.mp3 && mp3ConvertNeeded(song.ConvertedSongFileName, uint8(songConfig.mp3Quality)) {
		debrief.MP3Converted = true
		convertToMP3(song, songConfig.mp3Quality)
	}

	hasCover := processCover(&song, &songConfig, &debrief)
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
	serverURL = *subsonicUrl
	userName = *userNameFlag
	userPassword = *passwordFlag

	ffmpeg.LogCompiledCommand = false
	magick.Initialize()
	defer magick.Terminate()

	err := os.MkdirAll(*dir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	fmt.Println("Welcome to RockSonic!")
	var resp *http.Response
	if *playList != "nolist" {
		resp, err = http.Get(getUrl("getPlaylist", fmt.Sprintf("id=%s", *playList)))
	} else {
		resp, err = http.Get(getUrl("getStarred"))
	}
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var result SSResponse

	err = xml.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	songs := result.Starred.Songs
	if *playList != "nolist" {
		songs = result.Playlist.Songs
	}

	origDir := fmt.Sprintf("%s/.original", *dir)
	err = os.MkdirAll(origDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create original directory", err)
		panic(err)
	}

	convertedDir := fmt.Sprintf("%s/.converted", *dir)
	err = os.MkdirAll(convertedDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create converted directory", err)
		panic(err)
	}

	convertedSongDir := fmt.Sprintf("%s/songs", convertedDir)
	err = os.MkdirAll(convertedSongDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create converted songs directory", err)
		panic(err)
	}

	if *mp3 {
		result.Playlist.Name = fmt.Sprintf("%s_mp3", result.Playlist.Name)
	}
	combinedSongDir := fmt.Sprintf("%s/", *dir)
	if *playList != "nolist" {
		sanitized := utils.SanitizeFAT32Filename(result.Playlist.Name)
		combinedSongDir = fmt.Sprintf("%s/%s", combinedSongDir, sanitized)
	} else {
		combinedSongDir = fmt.Sprintf("%s/favs", combinedSongDir)
	}
	err = os.MkdirAll(combinedSongDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create converted songs directory", err)
		panic(err)
	}

	origSongDir := fmt.Sprintf("%s/songs", origDir)
	err = os.MkdirAll(origSongDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create original songs directory", err)
		panic(err)
	}

	origCoverDir := fmt.Sprintf("%s/covers", origDir)
	err = os.MkdirAll(origCoverDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create original cover directory", err)
		panic(err)
	}

	convertedCoverDir := fmt.Sprintf("%s/covers", convertedDir)
	err = os.MkdirAll(convertedCoverDir, os.ModePerm)
	if err != nil {
		fmt.Println("could not create converted cover directory", err)
		panic(err)
	}

	songConfig := SongConfig{
		mp3:               *mp3,
		flat:              *flat,
		coverSize:         *coverSize,
		convertedDir:      convertedDir,
		convertedSongDir:  convertedSongDir,
		combinedSongDir:   combinedSongDir,
		origCoverDir:      origCoverDir,
		origSongDir:       origSongDir,
		convertedCoverDir: convertedCoverDir,
		mp3Quality:        *mp3Quality,
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
		go processSong(song, songConfig, &wg, sem, deb)
		// fmt.Printf("%s -> %s -> %3s. %s\n", song.Artist, song.Album, song.Track, song.Title)
	}
	for i := 0; i < len(songs); i++ {
		debStr := <-deb
		fmt.Printf("%6d/%d %s\n", i+1, len(songs), debStr)
	}
	wg.Wait()
	fmt.Println(combinedSongDir)
}
