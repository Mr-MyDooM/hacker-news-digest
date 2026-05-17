package extractor

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/webp"
)

type WebImage struct {
	URL    string
	Width  int
	Height int
}

func FetchImage(srcURL, referrer, imageDir string) (*WebImage, error) {
	client := GetSafeClient(15 * time.Second)
	req, err := http.NewRequest("GET", srcURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if referrer != "" {
		req.Header.Set("Referer", referrer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch image: status %d", resp.StatusCode)
	}

	// Security: limit image download to 5MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}

	if len(data) < 100 {
		return nil, fmt.Errorf("image too small: %d bytes", len(data))
	}

	os.MkdirAll(imageDir, 0755)

	ct := resp.Header.Get("Content-Type")
	ext := detectExt(data, ct)
	hash := md5.Sum(data)
	filename := fmt.Sprintf("%x.%s", hash, ext)
	dest := filepath.Join(imageDir, filename)

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		os.WriteFile(dest, data, 0644)
	}

	img, _, err := decodeImage(data)
	if err != nil {
		log.Printf("Serving undecoded image %s (%s, %d bytes): %v", filename, ext, len(data), err)
		return &WebImage{URL: "/image/" + filename, Width: 0, Height: 0}, nil
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w < 100 || h < 100 {
		return nil, fmt.Errorf("image too small: %dx%d", w, h)
	}
	if w > 3*h || h > 3*w {
		return nil, fmt.Errorf("image bad aspect ratio: %dx%d", w, h)
	}
	if isMostlyWhite(img) {
		return nil, fmt.Errorf("image mostly white pixels")
	}

	return &WebImage{URL: "/image/" + filename, Width: w, Height: h}, nil
}

func decodeImage(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err == nil {
		return img, format, nil
	}
	if isWebP(data) {
		img, err := webp.Decode(bytes.NewReader(data))
		if err == nil {
			return img, "webp", nil
		}
	}
	return nil, "", err
}

func isWebP(data []byte) bool {
	return len(data) > 12 &&
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50
}

func detectExt(data []byte, contentType string) string {
	if len(data) < 4 {
		return extFromContentType(contentType, "png")
	}
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "jpg"
	}
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return "png"
	}
	if data[0] == 'G' && data[1] == 'I' && data[2] == 'F' {
		return "gif"
	}
	if data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		return "webp"
	}
	// AVIF: ISOBMFF container with ftyp box
	if len(data) > 12 && data[4] == 'f' && data[5] == 't' && data[6] == 'y' && data[7] == 'p' &&
		((data[8] == 'a' && data[9] == 'v' && data[10] == 'i' && data[11] == 'f') ||
			(data[8] == 'a' && data[9] == 'v' && data[10] == 'i' && data[11] == 's')) {
		return "avif"
	}
	// SVG: <?xml or <svg
	if len(data) > 4 && data[0] == '<' && (data[1] == 's' || data[1] == 'S') && (data[2] == 'v' || data[2] == 'V') && data[3] == 'g' {
		return "svg"
	}
	if len(data) > 5 && data[0] == '<' && data[1] == '?' && data[2] == 'x' && data[3] == 'm' && data[4] == 'l' {
		return "svg"
	}
	return extFromContentType(contentType, "png")
}

func extFromContentType(ct, fallback string) string {
	switch {
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return "jpg"
	case strings.Contains(ct, "png"):
		return "png"
	case strings.Contains(ct, "gif"):
		return "gif"
	case strings.Contains(ct, "webp"):
		return "webp"
	case strings.Contains(ct, "avif"):
		return "avif"
	case strings.Contains(ct, "svg"):
		return "svg"
	default:
		return fallback
	}
}

// isMostlyWhite checks if >99% of sampled pixels are near-white.
// Samples in a grid pattern (every 20th pixel) to avoid scanning the full image.
func isMostlyWhite(img image.Image) bool {
	bounds := img.Bounds()
	total := 0
	white := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 20 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 20 {
			r, g, b, _ := img.At(x, y).RGBA()
			// RGBA() returns 16-bit values (0-65535)
			if r>>8 > 250 && g>>8 > 250 && b>>8 > 250 {
				white++
			}
			total++
		}
	}
	if total == 0 {
		return false
	}
	return float64(white)/float64(total) > 0.99
}

func (w *WebImage) GetSizeStyle(maxWidth int) string {
	if w.Width == 0 || w.Height == 0 {
		return fmt.Sprintf("max-width:%dpx", maxWidth)
	}
	return fmt.Sprintf("max-width:%dpx;max-height:%dpx", maxWidth, maxWidth*w.Height/w.Width)
}
