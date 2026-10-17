package config

import "os"

type Config struct {
    Port   string
    DBPath string
}

// Load reads settings from environment variables, with sensible defaults.
func Load() Config {
    return Config{
        Port:   getEnv("PORT", "8080"),
        DBPath: getEnv("DB_PATH", "school.db"),
    }
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
