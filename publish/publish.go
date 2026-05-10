package publish

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/mj/hacker-news-digest/config"
	"github.com/mj/hacker-news-digest/hn"
	"github.com/mj/hacker-news-digest/template"
)

var cfg *config.Config

func Init(c *config.Config) {
	cfg = c
}

func GenFrontpage() {
	log.Println("Generating front page...")
	parser := hn.NewParser()
	newsList, err := parser.ParseNewsList()
	if err != nil {
		log.Fatalf("Failed to parse HN: %v", err)
	}

	log.Printf("Found %d stories", len(newsList))

	pullContentConcurrent(newsList)

	genPage(newsList, "index.html")
	genFeed(newsList, "feed.xml")
	genSitemap()
	genRobots()
}

func GenDaily(updatableDays int) {
	log.Printf("Refreshing daily pages for past %d days...", updatableDays)
	dailyItems, err := hn.GetDailyNews(updatableDays)
	if err != nil {
		log.Printf("Failed to get daily news: %v", err)
		return
	}

	for dateKey, items := range dailyItems {
		for i, item := range items {
			item.Rank = i + 1
		}
		pullContentConcurrent(items)
		path := filepath.Join("daily", dateKey, "index.html")
		genPage(items, path)
	}
}

// pullContentConcurrent fetches content and generates summaries for a list of news items concurrently.
// It uses a semaphore to limit the number of concurrent requests to avoid overwhelming external sites.
func pullContentConcurrent(newsList []*hn.News) {
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, news := range newsList {
		wg.Add(1)
		go func(n *hn.News) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			n.PullContent()
		}(news)
	}
	wg.Wait()
}

func genPage(newsList []*hn.News, path string) {
	if len(newsList) == 0 {
		return
	}

	dailyLinks := getDailyLinks()

	fullPath := filepath.Join(cfg.OutputDir, path)
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	pageURL := cfg.Site + "/" + path
	pageURL = filepath.Dir(pageURL) + "/"

	data := &template.PageData{
		NewsList:     newsList,
		LastUpdated:  time.Now().In(config.IST),
		Lang:         "en",
		DailyLinks:   dailyLinks,
		Path:         pageURL,
		Site:         cfg.Site,
		AdsenseID:    cfg.AdsenseID,
		DisableAds:   cfg.DisableAds,
		DisableTranslation: cfg.DisableTranslation,
	}

	rendered, err := template.Render(data)
	if err != nil {
		log.Printf("Template error for %s: %v", path, err)
		return
	}

	if err := os.WriteFile(fullPath, []byte(rendered), 0644); err != nil {
		log.Printf("Write error for %s: %v", path, err)
		return
	}

	log.Printf("Written %d bytes to %s", len(rendered), fullPath)
}

func getDailyLinks() []string {
	dailyDir := filepath.Join(cfg.OutputDir, "daily")
	entries, err := os.ReadDir(dailyDir)
	if err != nil {
		return nil
	}

	var links []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := time.Parse("2006-01-02", e.Name()); err == nil {
			links = append(links, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(links)))
	return links
}

func genFeed(newsList []*hn.News, filename string) {
	feed := template.RenderFeed(newsList, cfg.Site)
	fullPath := filepath.Join(cfg.OutputDir, filename)
	os.WriteFile(fullPath, []byte(feed), 0644)
	log.Printf("Generated feed.xml")
}

func genSitemap() {
	var sitemap string
	sitemap += "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
	sitemap += "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n"
	sitemap += "  <url><loc>" + cfg.Site + "/</loc><priority>1.0</priority></url>\n"

	dailyDir := filepath.Join(cfg.OutputDir, "daily")
	if entries, err := os.ReadDir(dailyDir); err == nil {
		var dates []string
		for _, e := range entries {
			if e.IsDir() {
				if _, err := time.Parse("2006-01-02", e.Name()); err == nil {
					dates = append(dates, e.Name())
				}
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dates)))
		for _, d := range dates {
			sitemap += "  <url><loc>" + cfg.Site + "/daily/" + d + "/</loc><priority>0.7</priority></url>\n"
		}
	}
	sitemap += "</urlset>\n"

	os.WriteFile(filepath.Join(cfg.OutputDir, "sitemap.xml"), []byte(sitemap), 0644)
	log.Println("Generated sitemap.xml")
}

func genRobots() {
	robots := `User-agent: *
Disallow: /static/
Disallow: /daily/
Allow: /
Crawl-delay: 10

User-agent: GPTBot
Disallow: /

User-agent: Claude-Web
Disallow: /

User-agent: CCBot
Disallow: /

User-agent: ChatGPT-User
Disallow: /

User-agent: Google-Extended
Disallow: /

User-agent: anthropic-ai
Disallow: /

Sitemap: ` + cfg.Site + "/sitemap.xml\n"
	os.WriteFile(filepath.Join(cfg.OutputDir, "robots.txt"), []byte(robots), 0644)
	log.Println("Generated robots.txt")
}


