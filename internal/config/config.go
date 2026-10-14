package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host          string
	Port          int
	AccessToken   string
	CommandPrefix string
	DatabasePath  string
	CPHistoryPath string
}

func Load() Config {
	loadDotEnv(".env")
	port, err := strconv.Atoi(env("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		log.Fatal("PORT must be an integer between 1 and 65535")
	}
	return Config{
		Host:          env("HOST", "127.0.0.1"),
		Port:          port,
		AccessToken:   os.Getenv("ONEBOT_V11_ACCESS_TOKEN"),
		CommandPrefix: env("COMMAND_START", "/"),
		DatabasePath:  env("QQBOT_DATABASE_PATH", "data/qqbot.db"),
		CPHistoryPath: os.Getenv("QQBOT_CP_HISTORY_PATH"),
	}
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), "\"'")
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
