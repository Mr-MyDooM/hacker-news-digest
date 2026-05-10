package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mj/hacker-news-digest/config"
	"github.com/mj/hacker-news-digest/db"
	"github.com/mj/hacker-news-digest/hn"
	"github.com/mj/hacker-news-digest/publish"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <home|daily>\n", os.Args[0])
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "home", "daily":
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s (use home or daily)\n", cmd)
		os.Exit(1)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg := config.Load()
	if err := db.Init(cfg); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	hn.Init(cfg)
	publish.Init(cfg)

	os.MkdirAll(cfg.ImageDir, 0755)

	if cmd == "home" {
		publish.GenFrontpage()
		n, err := db.ExpireSummaries()
		if err != nil {
			log.Printf("Summary expiry error: %v", err)
		} else {
			log.Printf("Evicted %d summaries", n)
		}
		if err := db.ExpireImages(); err != nil {
			log.Printf("Image expiry error: %v", err)
		}
	} else {
		publish.GenDaily(cfg.UpdatableDays)
	}

	copyStatic(cfg.OutputDir)
}

func copyStatic(outputDir string) {
	srcDir := "static"
	dstDir := filepath.Join(outputDir, "static")

	os.RemoveAll(dstDir)
	os.MkdirAll(dstDir, 0755)

	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		dst := filepath.Join(dstDir, rel)
		if info.IsDir() {
			os.MkdirAll(dst, 0755)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	})

	copyFile(filepath.Join(outputDir, "favicon.ico"), filepath.Join(outputDir, "static/favicon.ico"))
	copyFile(filepath.Join(outputDir, "feed"), filepath.Join(outputDir, "feed.xml"))
	copyFile(filepath.Join(outputDir, "hackernews"), filepath.Join(outputDir, "index.html"))
	copyFile(filepath.Join(outputDir, "ads.txt"), "static/ads.txt")
	copyFile(filepath.Join(outputDir, "sw.js"), "sw.js")

	hashStatic(outputDir)
}

func copyFile(dst, src string) {
	os.Remove(dst)
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	os.WriteFile(dst, data, 0644)
}

func hashStatic(outputDir string) {
	cssHash := md5Hash(filepath.Join(outputDir, "static/css/style.css"))
	jsHash := md5Hash(filepath.Join(outputDir, "static/js/hn.js"))

	if cssHash != "" {
		src := filepath.Join(outputDir, "static/css/style.css")
		dst := filepath.Join(outputDir, "static/css/style."+cssHash+".css")
		os.WriteFile(dst, readFile(src), 0644)
	}
	if jsHash != "" {
		src := filepath.Join(outputDir, "static/js/hn.js")
		dst := filepath.Join(outputDir, "static/js/hn."+jsHash+".js")
		os.WriteFile(dst, readFile(src), 0644)
	}

	replaceInHTML(outputDir, "style.css", "style."+cssHash+".css")
	replaceInHTML(outputDir, "hn.js", "hn."+jsHash+".js")
}

func md5Hash(path string) string {
	data := readFile(path)
	if data == nil {
		return ""
	}
	h := md5.Sum(data)
	return fmt.Sprintf("%x", h[:5])
}

func replaceInHTML(root, old, new string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		data := string(readFile(path))
		data = strings.ReplaceAll(data, old, new)
		os.WriteFile(path, []byte(data), 0644)
		return nil
	})
}

func readFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
