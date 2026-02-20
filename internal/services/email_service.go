package services

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService interface {
	SendVerificationEmail(to string, token string) error
	SendPasswordResetEmail(to string, token string) error
}

type emailService struct {
	dialer *gomail.Dialer
	sender string
}

func NewEmailService() EmailService {
	// On récupère les configs depuis le .env
	host := os.Getenv("SMTP_HOST")
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	sender := os.Getenv("SMTP_SENDER") // ex: no-reply@ton-saas.com

	dialer := gomail.NewDialer(host, port, user, password)

	return &emailService{
		dialer: dialer,
		sender: sender,
	}
}

func (s *emailService) SendVerificationEmail(to string, token string) error {
	link := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("FRONTEND_URL"), token)
	
	m := gomail.NewMessage()
	m.SetHeader("From", s.sender)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Vérifiez votre email - Mon Super SaaS")
	
	body := fmt.Sprintf(`
		<h1>Bienvenue !</h1>
		<p>Merci de vous être inscrit. Cliquez ci-dessous pour activer votre compte :</p>
		<p><a href="%s" style="padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">Vérifier mon email</a></p>
		<p>Ou copiez ce lien : %s</p>
	`, link, link)
	
	m.SetBody("text/html", body)

	return s.dialer.DialAndSend(m)
}

func (s *emailService) SendPasswordResetEmail(to string, token string) error {
	link := fmt.Sprintf("%s/reset-password?token=%s", os.Getenv("FRONTEND_URL"), token)

	m := gomail.NewMessage()
	m.SetHeader("From", s.sender)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Réinitialisation de mot de passe")

	body := fmt.Sprintf(`
		<h1>Mot de passe oublié ?</h1>
		<p>Pas de panique. Cliquez ci-dessous pour en définir un nouveau :</p>
		<p><a href="%s" style="padding: 10px 20px; background-color: #dc3545; color: white; text-decoration: none; border-radius: 5px;">Réinitialiser mon mot de passe</a></p>
		<p>Lien valide 15 minutes.</p>
	`, link)

	m.SetBody("text/html", body)

	return s.dialer.DialAndSend(m)
}