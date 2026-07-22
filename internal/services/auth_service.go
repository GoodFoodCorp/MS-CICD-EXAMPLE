package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ─── Interface ──────────────────────────────────────

type AuthService interface {
	RegisterUser(req *models.RegisterRequest) (string, error)
	LoginUser(req *models.LoginRequest) (string, string, error)
	LogoutUser(refreshToken string) error
	RefreshAccessToken(oldRefreshToken string) (string, string, error)
	VerifyEmail(token string) error
	ForgotPassword(req *models.ForgotPasswordRequest) (string, error)
	ResetPassword(req *models.ResetPasswordRequest) error
}

// ─── Implémentation ─────────────────────────────────

type authService struct {
	repo  repository.AuthRepository
	email EmailService
}

func NewAuthService(repo repository.AuthRepository, email EmailService) AuthService {
	return &authService{repo: repo, email: email}
}

func validatePasswordComplex(pass string) error {
	if len(pass) < 8 {
		return errors.New("le mot de passe doit faire au moins 8 caractères")
	}
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(pass)
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(pass)
	if !hasNumber || !hasLetter {
		return errors.New("le mot de passe doit contenir au moins une lettre et un chiffre")
	}
	return nil
}

func generateTokenString() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *authService) generateTokens(user *models.User) (string, string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	// Récupère les rôles de l'utilisateur
	roles, _ := s.repo.GetUserRoles(user.ID)
	roleNames := make([]string, len(roles))
	roleSlugs := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
		roleSlugs[i] = r.Slug
	}
	accessClaims := jwt.MapClaims{
		"sub":        user.ID,
		"email":      user.Email,
		"roles":      roleNames,
		"role_slugs": roleSlugs,
		"exp":        time.Now().Add(15 * time.Minute).Unix(),
	}
	// Si tenant_id est nil, on met une string vide dans le JWT
	if user.TenantID != nil {
		accessClaims["tenant_id"] = *user.TenantID
	} else {
		accessClaims["tenant_id"] = ""
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(secret)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{"sub": user.ID, "exp": time.Now().Add(7 * 24 * time.Hour).Unix()}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(secret)
	if err != nil {
		return "", "", err
	}

	rtModel := &models.RefreshToken{UserID: user.ID, Token: refreshToken, ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
	s.repo.CreateRefreshToken(rtModel)
	return accessToken, refreshToken, nil
}

func (s *authService) RegisterUser(req *models.RegisterRequest) (string, error) {
	if err := validatePasswordComplex(req.Password); err != nil {
		return "", err
	}

	// Vérifier si l'email existe déjà (globalement pour les utilisateurs sans tenant)
	existing, err := s.repo.FindGlobalByEmail(req.Email)
	if err == nil && existing != nil {
		return "", errors.New("cet email est déjà utilisé")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return "", err
	}

	// En dev (AUTO_VERIFY_EMAIL=true), on crée le compte déjà vérifié car
	// aucun SMTP n'est configuré — le client peut se connecter immédiatement.
	autoVerify := os.Getenv("AUTO_VERIFY_EMAIL") == "true"

	user := &models.User{
		Email:           req.Email,
		Password:        string(hashed),
		IsEmailVerified: autoVerify,
	}

	// Si un tenantID est fourni, on l'assigne (utilisateur tenant/restaurant)
	if req.TenantID != "" {
		user.TenantID = &req.TenantID
	}

	if err := s.repo.CreateUser(user); err != nil {
		return "", err
	}

	// Assigner le rôle "user" par défaut aux utilisateurs sans tenant (customers)
	if req.TenantID == "" {
		userRole, roleErr := s.repo.FindRoleBySlug("user")
		if roleErr == nil {
			s.repo.AssignRoleToUser(user.ID, userRole.ID)
		}
	}

	// Si l'email est déjà vérifié (mode dev), pas de token ni d'email à envoyer.
	if !autoVerify {
		token, _ := generateTokenString()
		emailToken := &models.EmailVerificationToken{
			UserID:    user.ID,
			Token:     token,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		s.repo.CreateEmailVerificationToken(emailToken)

		go func() {
			s.email.SendVerificationEmail(user.Email, token)
		}()
	}

	// Crée un profil vierge dans user-service (async, non bloquant)
	go NotifyUserServiceProfileCreation(user.ID)

	return user.ID, nil
}

func (s *authService) LoginUser(req *models.LoginRequest) (string, string, error) {
	user, err := s.repo.FindGlobalByEmail(req.Email)
	if err != nil {
		return "", "", errors.New("identifiants incorrects")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return "", "", errors.New("identifiants incorrects")
	}
	if !user.IsEmailVerified {
		return "", "", errors.New("veuillez vérifier votre email avant de vous connecter")
	}
	return s.generateTokens(user)
}

func (s *authService) LogoutUser(refreshToken string) error {
	return s.repo.RevokeRefreshToken(refreshToken)
}

func (s *authService) RefreshAccessToken(oldRefreshToken string) (string, string, error) {
	storedToken, err := s.repo.GetRefreshToken(oldRefreshToken)
	if err != nil {
		return "", "", errors.New("session expirée")
	}
	user, err := s.repo.FindUserByID(storedToken.UserID)
	if err != nil {
		return "", "", errors.New("user introuvable")
	}
	s.repo.RevokeRefreshToken(oldRefreshToken)
	return s.generateTokens(user)
}

func (s *authService) VerifyEmail(token string) error {
	vt, err := s.repo.FindEmailVerificationToken(token)
	if err != nil {
		return errors.New("token invalide ou expiré")
	}
	if err := s.repo.MarkUserAsVerified(vt.UserID); err != nil {
		return err
	}
	s.repo.DeleteEmailVerificationToken(vt.ID)
	return nil
}

func (s *authService) ForgotPassword(req *models.ForgotPasswordRequest) (string, error) {
	user, err := s.repo.FindGlobalByEmail(req.Email)
	if err != nil {
		return "", nil
	}
	token, _ := generateTokenString()
	pwdReset := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	s.repo.CreatePasswordResetToken(pwdReset)

	go func() {
		s.email.SendPasswordResetEmail(user.Email, token)
	}()

	return "", nil
}

func (s *authService) ResetPassword(req *models.ResetPasswordRequest) error {
	if req.NewPassword != req.ConfirmPassword {
		return errors.New("les mots de passe ne correspondent pas")
	}
	if err := validatePasswordComplex(req.NewPassword); err != nil {
		return err
	}
	pwdResetToken, err := s.repo.GetPasswordResetToken(req.Token)
	if err != nil {
		return errors.New("token invalide")
	}

	// Récupère l'utilisateur pour vérifier que le nouveau mot de passe n'est pas l'ancien
	user, err := s.repo.FindUserByID(pwdResetToken.UserID)
	if err != nil {
		return errors.New("utilisateur introuvable")
	}

	// Vérifie que le nouveau mot de passe n'est pas l'ancien
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)); err == nil {
		return errors.New("vous ne pouvez pas réutiliser votre ancien mot de passe")
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	s.repo.UpdateUserPassword(pwdResetToken.UserID, string(hashed))
	s.repo.MarkPasswordResetTokenAsUsed(req.Token)
	return nil
}
