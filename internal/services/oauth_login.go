package services

import (
	"auth-service/internal/models"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// LoginWithOAuth connecte (ou inscrit) un client à partir d'un profil déjà
// validé auprès du fournisseur, puis renvoie nos propres jetons : le reste de
// la plateforme ne connaît que le JWT Good Food.
//
// Trois cas :
//  1. le compte externe est déjà lié → connexion directe
//  2. un compte local existe avec le même email **vérifié** → rattachement
//  3. sinon → création d'un nouveau compte client
func (s *authService) LoginWithOAuth(profile *models.OAuthProfile) (string, string, error) {
	if profile == nil || profile.Subject == "" || profile.Email == "" {
		return "", "", errors.New("profil externe incomplet")
	}
	email := strings.ToLower(strings.TrimSpace(profile.Email))

	// 1. Compte externe déjà connu.
	if account, err := s.repo.FindOAuthAccount(profile.Provider, profile.Subject); err == nil && account != nil {
		user, err := s.repo.FindUserByID(account.UserID)
		if err != nil {
			return "", "", errors.New("compte introuvable")
		}
		return s.generateTokens(user)
	}

	// 2. Un compte local porte déjà cet email.
	if existing, err := s.repo.FindGlobalByEmail(email); err == nil && existing != nil && existing.ID != "" {
		// Rattacher sans email vérifié permettrait de prendre le contrôle d'un
		// compte existant en déclarant son adresse chez un fournisseur laxiste.
		if !profile.EmailVerified {
			return "", "", errors.New("cet email est déjà utilisé : connectez-vous avec votre mot de passe")
		}
		if err := s.linkAccount(existing.ID, profile, email); err != nil {
			return "", "", err
		}
		return s.generateTokens(existing)
	}

	// 3. Nouveau client.
	user, err := s.createOAuthUser(email)
	if err != nil {
		return "", "", err
	}
	if err := s.linkAccount(user.ID, profile, email); err != nil {
		return "", "", err
	}
	return s.generateTokens(user)
}

func (s *authService) linkAccount(userID string, profile *models.OAuthProfile, email string) error {
	return s.repo.CreateOAuthAccount(&models.OAuthAccount{
		UserID:         userID,
		Provider:       profile.Provider,
		ProviderUserID: profile.Subject,
		Email:          email,
	})
}

func (s *authService) createOAuthUser(email string) (*models.User, error) {
	// Le compte n'a pas de mot de passe utilisable : on stocke le hash d'un
	// secret aléatoire pour satisfaire la contrainte NOT NULL tout en rendant
	// la connexion par mot de passe impossible (personne ne connaît ce secret).
	filler := make([]byte, 32)
	if _, err := rand.Read(filler); err != nil {
		return nil, err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(filler)), 12)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:    email,
		Password: string(hashed),
		// L'email est garanti par le fournisseur : pas de vérification à refaire.
		IsEmailVerified: true,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	if role, err := s.repo.FindRoleBySlug("user"); err == nil {
		s.repo.AssignRoleToUser(user.ID, role.ID)
	}

	// Profil vierge côté user-service (asynchrone, non bloquant) — même
	// comportement que l'inscription classique.
	go NotifyUserServiceProfileCreation(user.ID)

	return user, nil
}
