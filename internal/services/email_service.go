package services

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

type EmailService interface {
	SendVerificationEmail(to string, token string) error
	SendPasswordResetEmail(to string, token string) error
}

type emailService struct {
	dialer *gomail.Dialer
	sender string
	// enabled is false without SMTP_HOST: nothing is sent and, outside
	// production only, the link is logged so local accounts stay usable.
	enabled    bool
	production bool
}

func NewEmailService() EmailService {
	host := os.Getenv("SMTP_HOST")
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	sender := os.Getenv("SMTP_SENDER")

	return &emailService{
		dialer:     gomail.NewDialer(host, port, user, password),
		sender:     sender,
		enabled:    host != "",
		production: os.Getenv("ENV") == "production",
	}
}

func (s *emailService) SendVerificationEmail(to string, token string) error {
	link := fmt.Sprintf("%s/verify-email?token=%s", frontendBaseURL(), url.QueryEscape(token))
	return s.send(to, "GoodFood — Confirmez votre adresse email", link, fmt.Sprintf(`
		<h1>Bienvenue chez GoodFood !</h1>
		<p>Confirmez votre adresse email pour activer votre compte :</p>
		<p><a href="%s" style="padding: 10px 20px; background-color: #1f4d3a; color: white; text-decoration: none; border-radius: 5px;">Confirmer mon email</a></p>
		<p>Ou copiez ce lien : %s</p>
		<p>Ce lien est valable 24 heures.</p>
	`, link, link), fmt.Sprintf("Bienvenue chez GoodFood !\n\nConfirmez votre adresse email : %s\n\nCe lien est valable 24 heures.", link))
}

func (s *emailService) SendPasswordResetEmail(to string, token string) error {
	link := fmt.Sprintf("%s/reset-password?token=%s", frontendBaseURL(), url.QueryEscape(token))
	return s.send(to, "GoodFood — Réinitialisation de votre mot de passe", link, fmt.Sprintf(`
		<h1>Mot de passe oublié ?</h1>
		<p>Cliquez ci-dessous pour choisir un nouveau mot de passe :</p>
		<p><a href="%s" style="padding: 10px 20px; background-color: #1f4d3a; color: white; text-decoration: none; border-radius: 5px;">Choisir un nouveau mot de passe</a></p>
		<p>Ce lien est valable 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez cet email.</p>
	`, link), fmt.Sprintf("Mot de passe oublié ?\n\nChoisissez un nouveau mot de passe : %s\n\nCe lien est valable 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez cet email.", link))
}

func (s *emailService) send(to, subject, link, html, text string) error {
	if !s.enabled {
		if s.production {
			return errors.New("SMTP_HOST is not configured")
		}
		log.Printf("[DEV] SMTP_HOST vide, email non envoyé à %s — %s : %s", to, subject, link)
		return nil
	}
	m := gomail.NewMessage()
	m.SetHeader("From", s.sender)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", text)
	m.AddAlternative("text/html", html)
	return s.dialer.DialAndSend(m)
}

func frontendBaseURL() string {
	if value := os.Getenv("FRONTEND_URL"); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "http://localhost:3000"
}
