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
		baseURL = "http://user-service:8087"
	}

	body, _ := json.Marshal(map[string]string{"user_id": userID})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/internal/profiles", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[WARN] Requête user-service invalide: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", os.Getenv("SERVICE_API_KEY"))
	resp, err := userHTTPClient.Do(req)
	if err != nil {
		log.Printf("[WARN] Impossible de créer le profil dans user-service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[WARN] user-service a retourné le status %d pour la création du profil", resp.StatusCode)
	}
}
