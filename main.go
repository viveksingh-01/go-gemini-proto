package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	genai "google.golang.org/genai"
)

// --- Types ---
type ChatRequest struct {
	UserID  string `json:"userId"`
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

// --- Global session map ---
var (
	client   *genai.Client
	sessions = make(map[string]*genai.Chat)
	mu       sync.Mutex // to protect sessions map
)

// --- Chat handler ---
func chatHandler(w http.ResponseWriter, r *http.Request) {
	// Decode JSON request
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mu.Lock()
	session, exists := sessions[req.UserID]
	if !exists {
		session, err := client.Chats.Create(r.Context(), "gemini-2.0-flash", nil, nil)
		if err != nil {
			log.Println(err)
		}
		sessions[req.UserID] = session
	}
	mu.Unlock()

	// Send user message
	resp, err := session.SendMessage(r.Context(), genai.Part{Text: req.Message})
	if err != nil {
		http.Error(w, fmt.Sprintf("Gemini error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println(resp.Text())

	// Respond with Gemini reply
	response := ChatResponse{Response: resp.Text()}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// --- Main ---
func main() {
	godotenv.Load()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	var err error
	client, err = genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	r := mux.NewRouter()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)
	r.HandleFunc("/chat", chatHandler).Methods("POST")

	fmt.Println("Server running on http://localhost:9090")
	log.Fatal(http.ListenAndServe(":9090", handler))
}
