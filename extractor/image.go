package extractor

import (
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WebImage struct {
	URL    string
	Width  int
	Height int
}

func FetchImage(srcURL, referrer, imageDir string) (*WebImage, error) {
	client := &http.Client{Timeout: 15 * time.Second}
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(data) < 100 {
		return nil, fmt.Errorf("image too small: %d bytes", len(data))
	}

	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w < 100 || h < 100 {
		return nil, fmt.Errorf("image too small: %dx%d", w, h)
	}
	if w > 3*h || h > 3*w {
		return nil, fmt.Errorf("image bad aspect ratio: %dx%d", w, h)
	}

	os.MkdirAll(imageDir, 0755)

	ext := detectExt(data)
	hash := md5.Sum(data)
	filename := fmt.Sprintf("%x.%s", hash, ext)
	dest := filepath.Join(imageDir, filename)

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		os.WriteFile(dest, data, 0644)
	}

	return &WebImage{URL: "/image/" + filename, Width: w, Height: h}, nil
}

func detectExt(data []byte) string {
	if len(data) < 4 {
		return "png"
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
	return "png"
}

func (w *WebImage) GetSizeStyle(maxWidth int) string {
	if w.Width == 0 || w.Height == 0 {
		return fmt.Sprintf("max-width:%dpx", maxWidth)
	}
	return fmt.Sprintf("max-width:%dpx;max-height:%dpx", maxWidth, maxWidth*w.Height/w.Width)
}
