package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"helixrun/internal/agents"
	httpserver "helixrun/internal/http"
)

func main() {
	// .env laden (optioneel, errors negeren als er geen .env is)
	_ = godotenv.Load()

	configDir := os.Getenv("HELIXRUN_CONFIG_DIR")
	if configDir == "" {
		configDir = "./configs/agents"
	}

	addr := os.Getenv("HELIXRUN_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	reg, err := agents.LoadRegistry(configDir)
	if err != nil {
		log.Fatalf("failed to load agent registry: %v", err)
	}

	log.Printf("Loaded agents: %v", reg.ListAgentIDs())

	mux := http.NewServeMux()

	chatServer := httpserver.NewChatServer(reg)
	mux.HandleFunc("/chat", chatServer.ChatHandler)

	fileServer := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fileServer)

	log.Printf("HelixRun starter listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
