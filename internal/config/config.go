package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Supabase SupabaseConfig
}

type ServerConfig struct {
	Addr string
}

type SupabaseConfig struct {
	URL            string
	AnonKey        string
	ServiceRoleKey string
}

func Load() (Config, error) {
	for _, path := range []string{".env", "../.env", "../../.env"} {
		_ = godotenv.Load(path)
	}

	anonKey := firstNonEmpty(os.Getenv("SUPABASE_ANON_KEY"), os.Getenv("SUPABASE_KEY"))
	cfg := Config{
		Server: ServerConfig{
			Addr: envOrDefault("SERVER_ADDR", ":8080"),
		},
		Supabase: SupabaseConfig{
			URL:            os.Getenv("SUPABASE_URL"),
			AnonKey:        anonKey,
			ServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		},
	}

	if cfg.Supabase.URL == "" || cfg.Supabase.AnonKey == "" {
		return Config{}, fmt.Errorf("请在 .env 中设置 SUPABASE_URL 和 SUPABASE_ANON_KEY（与 Flutter my_ai_project 一致）")
	}
	return cfg, nil
}

func (s SupabaseConfig) DBKey() string {
	if s.ServiceRoleKey != "" {
		return s.ServiceRoleKey
	}
	return s.AnonKey
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
