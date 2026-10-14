package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"qqbot-go/internal/bot"
	"qqbot-go/internal/config"
	"qqbot-go/internal/onebot"
	"qqbot-go/internal/store"
)

func main() {
	cfg := config.Load()
	client := onebot.NewClient()
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.DB.Close()
	if cfg.CPHistoryPath != "" {
		count, err := db.ImportLegacyPairHistory(cfg.CPHistoryPath)
		if err != nil {
			log.Fatalf("import legacy CP history: %v", err)
		}
		log.Printf("imported %d legacy CP history entries", count)
	}
	handler := bot.NewHandler(client, cfg.CommandPrefix, db)
	server := onebot.NewServer(cfg.AccessToken, client, handler.HandleEvent)

	mux := http.NewServeMux()
	mux.Handle("/onebot/v11/ws", server)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	log.Printf("QQ Bot Go listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
