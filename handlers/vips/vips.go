package vips

import (
	"bytes"
	"io"

	"path"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/qor/media"
	"github.com/theplant/bimg"
)

var (
	EnableGenerateWebp = false
	WebpQuality        = 85
	JPEGQuality        = 80
	PNGQuality         = 90
	PNGCompression     = 9
)

type Config struct {
	EnableGenerateWebp bool
	WebpQuality        int
	JPEGQuality        int
	PNGQuality         int
	PNGCompression     int
}

type bimgImageHandler struct{}

func (bimgImageHandler) CouldHandle(media media.Media) bool {
	return media.IsImage()
}

func (bimgImageHandler) Handle(media media.Media, file media.FileInterface, option *media.Option) (err error) {
	// Crop & Resize
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, file); err != nil {
		return err
	}
	quality := getQualityByImageType(media.URL())

	// Save Original Image
	{
		img := copyImage(buffer.Bytes())
		bimgOption := bimg.Options{Quality: quality, Palette: true, Compression: PNGCompression}
		// Process & Save original image
		if buf, err := img.Process(bimgOption); err == nil {
			media.Store(media.URL("original"), option, bytes.NewReader(buf))
		} else {
			return err
		}
		if err = generateWebp(media, option, bimgOption, buffer.Bytes(), "original"); err != nil {
			return err
		}
	}

	// Handle default image
	{
		img := copyImage(buffer.Bytes())
		bimgOption := bimg.Options{Quality: quality, Palette: true, Compression: PNGCompression}
		// Crop original image if specified
		if cropOption := media.GetCropOption("original"); cropOption != nil {
			bimgOption.Top = cropOption.Min.Y
			bimgOption.Left = cropOption.Min.X
			bimgOption.AreaWidth = cropOption.Max.X - cropOption.Min.X
			bimgOption.AreaHeight = cropOption.Max.Y - cropOption.Min.Y
		}

		if buf, err := img.Process(bimgOption); err == nil {
			if err = media.Store(media.URL(), option, bytes.NewReader(buf)); err != nil {
				return err
			}
			if err = generateWebp(media, option, bimgOption, buf); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Handle size images
	for key, size := range media.GetSizes() {
		if key == "original" {
			continue
		}
		img := copyImage(buffer.Bytes())
		if cropOption := media.GetCropOption(key); cropOption != nil {
			if _, err := img.Extract(cropOption.Min.Y, cropOption.Min.X, cropOption.Max.X-cropOption.Min.X, cropOption.Max.Y-cropOption.Min.Y); err != nil {
				return err
			}
		}
		bimgOption := bimg.Options{
			Width:       size.Width,
			Height:      size.Height,
			Quality:     quality,
			Compression: PNGCompression,
			Palette:     true,
			Enlarge:     true,
		}
		// Process & Save size image
		if buf, err := img.Process(bimgOption); err == nil {
			if err = media.Store(media.URL(key), option, bytes.NewReader(buf)); err != nil {
				return err
			}
			if err = generateWebp(media, option, bimg.Options{}, buf, key); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return
}

func generateWebp(media media.Media, option *media.Option, bimgOption bimg.Options, buffer []byte, size ...string) (err error) {
	if !EnableGenerateWebp {
		return
	}
	img := copyImage(buffer)
	bimgOption.Type = bimg.WEBP
	bimgOption.Quality = WebpQuality
	if buf, err := img.Process(bimgOption); err == nil {
		url := media.URL(size...)
		ext := path.Ext(url)
		extArr := strings.Split(ext, "?")
		i := strings.LastIndex(url, ext)
		webpUrl := url[:i] + strings.Replace(url[i:], extArr[0], ".webp", 1)
		media.Store(webpUrl, option, bytes.NewReader(buf))
	} else {
		return err
	}
	return
}

func copyImage(buffer []byte) (img *bimg.Image) {
	bs := make([]byte, len(buffer))
	copy(bs, buffer)
	img = bimg.NewImage(bs)
	return
}

func getQualityByImageType(url string) int {
	imgType, err := media.GetImageFormat(url)
	if err != nil {
		return 0
	}
	switch *imgType {
	case imaging.JPEG:
		return JPEGQuality
	case imaging.PNG:
		return PNGQuality
	}
	return 0
}

func UseVips(cfg Config) {
	if cfg.EnableGenerateWebp {
		EnableGenerateWebp = true
	}
	if cfg.WebpQuality > 0 && cfg.WebpQuality <= 100 {
		WebpQuality = cfg.WebpQuality
	}
	if cfg.JPEGQuality > 0 && cfg.JPEGQuality <= 100 {
		JPEGQuality = cfg.JPEGQuality
	}
	if cfg.PNGQuality > 0 && cfg.PNGQuality <= 100 {
		WebpQuality = cfg.WebpQuality
	}
	if cfg.PNGCompression > 0 && cfg.PNGCompression <= 9 {
		PNGCompression = cfg.PNGCompression
	}
	media.RegisterMediaHandler("image_handler", bimgImageHandler{})
}
