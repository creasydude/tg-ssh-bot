package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	encKeyHex := os.Getenv("ENCRYPTION_KEY")
	if encKeyHex == "" {
		log.Fatal("ENCRYPTION_KEY is required (64 hex chars)")
	}

	key, err := ParseKey(encKeyHex)
	if err != nil {
		log.Fatalf("Invalid ENCRYPTION_KEY: %v", err)
	}

	var allowed []int64
	if s := os.Getenv("ALLOWED_USERS"); s != "" {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				var id int64
				if id, err = strconv.ParseInt(part, 10, 64); err == nil {
					allowed = append(allowed, id)
				}
			}
		}
	}

	initDB("bot.db")
	defer db.Close()

	bot := NewBot(token, key, allowed)
	bot.Start()
}
