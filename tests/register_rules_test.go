package tests

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// The CNIL rule for a rate-limited login: 8+ characters, three kinds out of four.
func TestRegister_PasswordRule(t *testing.T) {
	for _, tc := range []struct {
		password string
		ok       bool
	}{
		{"motdepasse", false},  // one kind
		{"motdepasse1", false}, // two kinds
		{"Motdepasse1", true},
		{"motdepasse1!", true},
		{"Électrique-été", true},
		{"Ab1!", false},
		{string(bytes.Repeat([]byte("Aa1"), 25)), false}, // 75 bytes: bcrypt would cut it
	} {
		repo := new(MockAuthRepository)
		repo.On("FindGlobalByEmail", mock.Anything).Return(nil, errors.New("not found")).Maybe()
		// Past the rule, the account creation is cut short: only the rule is tested.
		repo.On("CreateUser", mock.Anything).Return(errors.New("stop")).Maybe()
		_, err := services.NewAuthService(repo, new(MockEmailService)).RegisterUser(&models.RegisterRequest{Email: "a@b.fr", Password: tc.password})
		var refused services.InputError
		assert.Equal(t, !tc.ok, errors.As(err, &refused), "%q: %v", tc.password, err)
	}
}

// One account per address whatever the case: the lookup uses the normalized email.
func TestLogin_EmailIsNormalized(t *testing.T) {
	repo := new(MockAuthRepository)
	repo.On("FindGlobalByEmail", "camille@example.com").Return(nil, errors.New("not found")).Once()
	_, _, err := services.NewAuthService(repo, new(MockEmailService)).LoginUser(&models.LoginRequest{Email: "  Camille@Example.COM ", Password: "x"})
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestRegister_AnswersInPlainFrench(t *testing.T) {
	for _, tc := range []struct {
		body, want string
		status     int
	}{
		{`{"email":"a@b.fr","password":"abc"}`, "au moins 8 caractères", http.StatusBadRequest},
		{`{"email":"pas-un-email","password":"Motdepasse1"}`, "email n'est pas valide", http.StatusBadRequest},
		{`{"password":"Motdepasse1"}`, "Merci de renseigner l'adresse email", http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		SetupAuthRouter(new(MockAuthService)).ServeHTTP(w, req)
		var answer map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &answer)
		assert.Equal(t, tc.status, w.Code, tc.body)
		assert.Contains(t, answer["error"], tc.want, tc.body)
	}

	weak := new(MockAuthService)
	weak.On("RegisterUser", mock.Anything).Return("", services.InputError("le mot de passe doit mêler au moins trois types de caractères")).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"a@b.fr","password":"motdepasse1"}`))
	req.Header.Set("Content-Type", "application/json")
	SetupAuthRouter(weak).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "a weak password is the user's input, not a conflict")
}
