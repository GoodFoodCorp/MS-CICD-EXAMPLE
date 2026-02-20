package services

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

var userHTTPClient = &http.Client{Timeout: 5 * time.Second}

// NotifyUserServiceProfileCreation appelle user-service pour créer un profil vierge
// après l'inscription. Appel non-bloquant (goroutine recommandée).
func NotifyUserServiceProfileCreation(userID string) {
	baseURL := os.Getenv("USER_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://user-service:8082"
	}

	body, _ := json.Marshal(map[string]string{"user_id": userID})
	resp, err := userHTTPClient.Post(baseURL+"/internal/profiles", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[WARN] Impossible de créer le profil dans user-service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[WARN] user-service a retourné le status %d pour la création du profil", resp.StatusCode)
	}
}
