package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Restaurant est un restaurant (franchise) tel qu'exposé par franchise-service,
// qui en est désormais le propriétaire. auth-service ne stocke plus que
// l'identifiant du restaurant sur l'utilisateur (pas de jointure).
type Restaurant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive bool   `json:"isActive"`
}

var franchiseHTTPClient = &http.Client{Timeout: 5 * time.Second}

// FetchRestaurantsBySlug interroge franchise-service et indexe les restaurants
// par slug. Utilisé par le seeder pour rattacher les managers à leur restaurant.
func FetchRestaurantsBySlug() (map[string]Restaurant, error) {
	baseURL := os.Getenv("FRANCHISE_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://franchise-service:8089"
	}

	resp, err := franchiseHTTPClient.Get(baseURL + "/api/restaurants")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("franchise-service a retourné le status %d", resp.StatusCode)
	}

	var restaurants []Restaurant
	if err := json.NewDecoder(resp.Body).Decode(&restaurants); err != nil {
		return nil, err
	}

	bySlug := make(map[string]Restaurant, len(restaurants))
	for _, r := range restaurants {
		bySlug[r.Slug] = r
	}
	return bySlug, nil
}
