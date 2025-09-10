package lib

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

var serverURL = ""
var serverUser = ""
var serverPassword = ""

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
	Songs []Song `xml:"song"`
}
type Playlist struct {
	Name  string `xml:"name,attr"`
	Songs []Song `xml:"entry"`
}
type Album struct {
	Artist string `xml:"artist,attr"`
}

type SubPlaylist struct {
	XMLName  xml.Name `xml:"subsonic-response"`
	Playlist Playlist `xml:"playlist"`
}
type SubStarred struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Starred Starred  `xml:"starred"`
}
type SubAlbum struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Album   Album    `xml:"album"`
}

func GetUrl(endpoint string, extraParams ...string) string {
	extraString := ""
	for _, param := range extraParams {
		extraString = fmt.Sprintf("%s&%s", extraString, param)
	}
	url := fmt.Sprintf("%s/%s?v=1.16.1&c=rocksonic&u=%s&p=%s%s", serverURL, endpoint, serverUser, serverPassword, extraString)
	// fmt.Printf("%s/%s?v=1.16.1&c=rocksonic&u=%s&p=%s%s\n", serverURL, endpoint, userName, "*****", extraString)
	return url
}

func GetAlbum(id string) (Album, error) {
	var resp *http.Response
	resp, err := http.Get(GetUrl("getAlbum", fmt.Sprintf("id=%s", id)))
	var albumResult SubAlbum

	err = xml.NewDecoder(resp.Body).Decode(&albumResult)
	if err != nil {
		return Album{}, err
	}
	return albumResult.Album, nil
}
func GetStarred() ([]Song, error) {
	var resp *http.Response
	resp, err := http.Get(GetUrl("getStarred"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SubStarred

	err = xml.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result.Starred.Songs, nil
}
func GetPlaylist(playlist string) ([]Song, string, error) {
	var resp *http.Response
	resp, err := http.Get(GetUrl("getPlaylist", fmt.Sprintf("id=%s", playlist)))
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var result SubPlaylist

	err = xml.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, "", err
	}
	return result.Playlist.Songs, result.Playlist.Name, nil
}
func DownloadSong(song Song) error {

	url := GetUrl("download", "id="+string(song.ID))
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
func SetServer(url string, username string, password string) {
	serverURL = url
	serverUser = username
	serverPassword = password
}
