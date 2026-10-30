package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"auth-service/internal/tokens"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"regexp"
	"time"

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
	ChangePassword(userID string, req *models.ChangePasswordRequest) error
	DeleteAccount(userID string, req *models.DeleteAccountRequest) error
	UpgradeToDelivery(userID string) error
	LoginWithOAuth(profile *models.OAuthProfile) (string, string, error)
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

// generateTokens issues a fresh access/refresh pair and stores the refresh
// token's hash so it can be rotated and revoked.
func (s *authService) generateTokens(user *models.User) (string, string, error) {
	cfg, err := tokens.FromEnv()
	if err != nil {
		return "", "", err
	}
	roles, _ := s.repo.GetUserRoles(user.ID)
	roleNames := make([]string, len(roles))
	roleSlugs := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
		roleSlugs[i] = r.Slug
	}
	tenantID := ""
	if user.TenantID != nil {
		tenantID = *user.TenantID
	}
	now := time.Now()
	accessToken, err := cfg.IssueAccess(user.ID, map[string]any{
		"email":      user.Email,
		"roles":      roleNames,
		"role_slugs": roleSlugs,
		"tenant_id":  tenantID,
	}, now)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := cfg.IssueRefresh(user.ID, now)
	if err != nil {
		return "", "", err
	}
	stored := &models.RefreshToken{UserID: user.ID, Token: tokens.Hash(refreshToken), ExpiresAt: now.Add(tokens.RefreshTTL)}
	if err := s.repo.CreateRefreshToken(stored); err != nil {
		return "", "", err
	}
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
			if err := s.email.SendVerificationEmail(user.Email, token); err != nil {
				log.Printf("[ERROR] Email de vérification non envoyé à %s: %v", user.Email, err)
			}
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
	return s.repo.RevokeRefreshToken(tokens.Hash(refreshToken))
}

// RefreshAccessToken rotates a refresh token: the presented one must be a
// genuine, unexpired, unrevoked refresh token (an access token is refused), and
// it is revoked before a new pair is issued.
func (s *authService) RefreshAccessToken(oldRefreshToken string) (string, string, error) {
	cfg, err := tokens.FromEnv()
	if err != nil {
		return "", "", err
	}
	subject, err := cfg.ParseRefresh(oldRefreshToken)
	if err != nil {
		return "", "", errors.New("session expirée")
	}
	hash := tokens.Hash(oldRefreshToken)
	storedToken, err := s.repo.GetRefreshToken(hash)
	if err != nil || storedToken.UserID != subject {
		return "", "", errors.New("session expirée")
	}
	user, err := s.repo.FindUserByID(storedToken.UserID)
	if err != nil {
		return "", "", errors.New("user introuvable")
	}
	if err := s.repo.RevokeRefreshToken(hash); err != nil {
		return "", "", err
	}
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
		if err := s.email.SendPasswordResetEmail(user.Email, token); err != nil {
			log.Printf("[ERROR] Email de réinitialisation non envoyé à %s: %v", user.Email, err)
		}
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

func (s *authService) UpgradeToDelivery(userID string) error {
	role, err := s.repo.FindRoleBySlug("livreur")
	if err != nil {
		return errors.New("rôle livreur introuvable")
	}
	return s.repo.AssignRoleToUser(userID, role.ID)
}

// ChangePassword lets an authenticated user set a new password, verifying the
// current one first. All existing sessions are revoked afterwards.
func (s *authService) ChangePassword(userID string, req *models.ChangePasswordRequest) error {
	if req.NewPassword != req.ConfirmPassword {
		return errors.New("les mots de passe ne correspondent pas")
	}
	if err := validatePasswordComplex(req.NewPassword); err != nil {
		return err
	}
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("utilisateur introuvable")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return errors.New("mot de passe actuel incorrect")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)); err == nil {
		return errors.New("vous ne pouvez pas réutiliser votre ancien mot de passe")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUserPassword(userID, string(hashed)); err != nil {
		return err
	}
	return s.repo.RevokeAllRefreshTokensForUser(userID)
}

// DeleteAccount soft-deletes the caller's own account after verifying their
// password, and revokes every active session.
func (s *authService) DeleteAccount(userID string, req *models.DeleteAccountRequest) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("utilisateur introuvable")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return errors.New("mot de passe incorrect")
	}
	if err := s.repo.RevokeAllRefreshTokensForUser(userID); err != nil {
		return err
	}
	if err := s.repo.DeleteUser(userID); err != nil {
		return err
	}
	// Right to erasure: the personal data held by user-service goes too.
	go NotifyUserServiceErasure(userID)
	return nil
}
