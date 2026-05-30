package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisAddr     string
	RedisQueueKey string

	SeaweedFSMaster string
	SeaweedFSFiler  string

	ServiceKey     string
	HLSTokenSecret string
	PublicHost     string

	APIPort       string
	ReadPublic    bool // true = GET endpoints require no auth
	CORSOrigins   []string
	MaxUploadSize int64 // bytes

	SpriteIntervalSeconds int
	SpriteWidth           int
	SpriteHeight          int
	SpriteColumns         int
}

func Load() (*Config, error) {
	var missing []string
	req := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &Config{
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          req("POSTGRES_USER"),
		DBPassword:      req("POSTGRES_PASSWORD"),
		DBName:          req("POSTGRES_DB"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisQueueKey:   getEnv("REDIS_QUEUE_KEY", "transcoding_queue"),
		SeaweedFSMaster: getEnv("SEAWEEDFS_MASTER", "http://localhost:9333"),
		SeaweedFSFiler:  getEnv("SEAWEEDFS_FILER", "http://localhost:8888"),
		ServiceKey:      req("SERVICE_KEY"),
		HLSTokenSecret:  req("HLS_TOKEN_SECRET"),
		APIPort:         getEnv("API_PORT", "8000"),
		ReadPublic:      getEnv("READ_PUBLIC", "true") != "false",
		CORSOrigins:     parseCORSOrigins(getEnv("CORS_ORIGINS", "*")),
		MaxUploadSize:   getEnvInt64("MAX_UPLOAD_SIZE_GB", 50) << 30,

		SpriteIntervalSeconds: int(getEnvInt64("SPRITE_INTERVAL_SECONDS", 10)),
		SpriteWidth:           int(getEnvInt64("SPRITE_WIDTH", 320)),
		SpriteHeight:          int(getEnvInt64("SPRITE_HEIGHT", 180)),
		SpriteColumns:         int(getEnvInt64("SPRITE_COLUMNS", 10)),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("required environment variables not set: %s", strings.Join(missing, ", "))
	}

	cfg.PublicHost = resolvePublicHost()
	return cfg, nil
}

func resolvePublicHost() string {
	if h := os.Getenv("PUBLIC_HOST"); h != "" {
		return strings.TrimRight(h, "/")
	}
	hlsURL := os.Getenv("PUBLIC_HLS_URL")
	u, err := url.Parse(hlsURL)
	if err == nil && u.Host != "" {
		return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}
	s := strings.TrimRight(hlsURL, "/")
	return strings.TrimRight(strings.TrimSuffix(s, "/hls"), "/")
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseCORSOrigins(s string) []string {
	var origins []string
	for _, o := range strings.Split(s, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}
