package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var IST = time.FixedZone("IST", 5*60*60+30*60)

type Config struct {
	Debug bool
	Site  string

	OpenAIAPIKey string
	OpenAIBase   string
	OpenAIKeys   []string
	OpenAIModel  string
	OpenAIScore  int

	GeminiAPIKey  string
	GeminiModel   string
	GeminiScore   int
	DisableGemini bool

	OpenRouterAPIKey  string
	OpenRouterModels  []string
	OpenRouterScore   int
	DisableOpenRouter bool

	AdsenseID string

	DisableLLaMA        bool
	DisableTransformer  bool
	DisableTranslation  bool
	DisableAds          bool
	DisableSummaryCache bool
	ForceFetchImage     bool

	SummaryTTL    int
	SummarySize   int
	UpdatableDays int
	LocalLLMScore int
	LLMRate       int

	OutputDir string
	ImageDir  string
	DBPath    string
}

func intEnv(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func Load() *Config {
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		wd, _ := os.Getwd()
		dbPath = "sqlite:///" + filepath.Join(wd, "hackernews.db")
	}
	dbPath = strings.TrimPrefix(dbPath, "sqlite:///")

	wd, _ := os.Getwd()
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = filepath.Join(wd, "output")
	}

	keysStr := os.Getenv("OPENAI_API_KEY")
	var keys []string
	if keysStr != "" {
		keys = strings.Split(keysStr, ",")
	}

	site := os.Getenv("SITE_URL")
	if site == "" {
		site = "https://HackerNews.mrityunjay.dev"
	}

	cfg := &Config{
		Debug:  os.Getenv("DEBUG") == "1",
		Site:   site,
		DBPath: dbPath,

		OpenAIAPIKey: keysStr,
		OpenAIBase:   os.Getenv("OPENAI_API_BASE"),
		OpenAIKeys:   keys,
		OpenAIModel:  os.Getenv("OPENAI_MODEL"),
		OpenAIScore:  intEnv("OPENAI_SCORE_THRESHOLD", 20),

		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		GeminiModel:   os.Getenv("GEMINI_MODEL"),
		GeminiScore:   intEnv("GEMINI_SCORE_THRESHOLD", 20),
		DisableGemini: os.Getenv("DISABLE_GEMINI") == "1",

		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModels:  parseModels(os.Getenv("OPENROUTER_MODELS")),
		OpenRouterScore:   intEnv("OPENROUTER_SCORE_THRESHOLD", 1),
		DisableOpenRouter: os.Getenv("DISABLE_OPENROUTER") == "1",

		DisableLLaMA:       os.Getenv("DISABLE_LLAMA") == "1",
		DisableTransformer: os.Getenv("DISABLE_TRANSFORMER") == "1",
		AdsenseID:          os.Getenv("ADSENSE_ID"),

		DisableTranslation:  os.Getenv("DISABLE_TRANSLATION") == "1",
		DisableAds:          os.Getenv("DISABLE_ADS") == "1",
		DisableSummaryCache: os.Getenv("DISABLE_SUMMARY_CACHE") == "1",
		ForceFetchImage:     os.Getenv("FORCE_FETCH_FEATURE_IMAGE") == "1",

		SummaryTTL:    intEnv("SUMMARY_TTL_DAYS", 60) * 86400,
		SummarySize:   400,
		UpdatableDays: intEnv("UPDATABLE_WITHIN_DAYS", 15),
		LocalLLMScore: 10,
		LLMRate:       intEnv("LLM_RATE_LIMIT", 30),

		OutputDir: outputDir,
		ImageDir:  filepath.Join(outputDir, "image"),
	}

	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = "gpt-3.5-turbo"
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = "gemini-3.1-flash-lite"
	}

	if cfg.OpenAIBase == "" {
		cfg.OpenAIBase = "https://api.openai.com/v1"
	}

	log.Printf("Config: site=%s db=%s openai_model=%s gemini_model=%s openrouter_models=%v",
		cfg.Site, cfg.DBPath, cfg.OpenAIModel, cfg.GeminiModel, cfg.OpenRouterModels)
	return cfg
}

func parseModels(s string) []string {
	if s == "" {
		return nil
	}
	var models []string
	for _, m := range strings.Split(s, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			models = append(models, m)
		}
	}
	return models
}
