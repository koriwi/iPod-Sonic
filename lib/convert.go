package lib

import (
	"errors"
	"fmt"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	magick "gopkg.in/gographics/imagick.v3/imagick"
)

func ExtractCover(song Song, coverSize uint) error {

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
func InitMagick() {
	magick.Initialize()
}
func TerminateMagick() {
	magick.Terminate()
}
func ConvertToMP3(song Song, quality uint) error {
	err := ffmpeg.Input(song.OriginalSongFileName).Output(
		song.ConvertedSongFileName,
		ffmpeg.KwArgs{"vn": "", "q:a": quality, "map_metadata": 0, "id3v2_version": 3, "write_id3v1": 1, "write_id3v2": 1, "metadata": fmt.Sprintf("rocksonic_quality=%d", quality), "y": ""},
	).OverWriteOutput().Run()
	if err != nil {
		return errors.New(fmt.Sprint("error extracting cover from", song.OriginalSongFileName, err))
	}

	return nil
}
