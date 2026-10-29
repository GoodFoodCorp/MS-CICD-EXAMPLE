package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var userHTTPClient = &http.Client{Timeout: 5 * time.Second}

// erasureAttempts bounds the retries of a user data erasure.
const erasureAttempts = 5

func userServiceURL() string {
	if base := os.Getenv("USER_SERVICE_URL"); base != "" {
		return base
	}
	return "http://user-service:8087"
}

// callUserService sends one internal call to user-service (X-Service-Key).
func callUserService(method, path string, body []byte) error {
	req, err := http.NewRequest(method, userServiceURL()+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", os.Getenv("SERVICE_API_KEY"))
	resp, err := userHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("user-service a retourné le status %d", resp.StatusCode)
	}
	return nil
}

// NotifyUserServiceProfileCreation appelle user-service pour créer un profil vierge
// après l'inscription. Appel non-bloquant (goroutine recommandée) : le profil
// est sinon créé à la volée au premier accès.
func NotifyUserServiceProfileCreation(userID string) {
	body, _ := json.Marshal(map[string]string{"user_id": userID})
	if err := callUserService(http.MethodPost, "/internal/profiles", body); err != nil {
		log.Printf("[WARN] Impossible de créer le profil dans user-service: %v", err)
	}
}

// NotifyUserServiceErasure demande à user-service d'effacer les données
// personnelles d'un compte supprimé (droit à l'effacement), avec quelques
// nouvelles tentatives espacées. Non bloquant (goroutine).
func NotifyUserServiceErasure(userID string) {
	path := "/internal/profiles/" + url.PathEscape(userID)
	delay := time.Second
	for attempt := 1; attempt <= erasureAttempts; attempt++ {
		err := callUserService(http.MethodDelete, path, nil)
		if err == nil {
			return
		}
		log.Printf("[WARN] Effacement RGPD dans user-service (essai %d/%d): %v", attempt, erasureAttempts, err)
		time.Sleep(delay)
		delay *= 2
	}
	log.Printf("[ERROR] Données de l'utilisateur %s non effacées dans user-service: à traiter manuellement", userID)
}
