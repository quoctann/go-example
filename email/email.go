package email

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"go-example/config"
	"go-example/utils"
	"html/template"
	"log"
	"math/big"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

/*
	Dev resources:
		mailtrap.io
		brevo.com
		smtp4dev - docker run -p 3000:80 -p 1025:1025 rnwood/smtp4dev
		https://github.com/wneessen/go-mail

	Notes:
		Implement với queue system (redis, rabbitmq) để gửi mail hàng loạt
		Track email metrics
		Retry mechanism
		Tuân thủ GDPR, CAN-SPAM về unsubscribe
*/

type IEmailService interface {
	// Send single email to recipient
	Send(to, subject, body string) error
}

type EmailService struct {
	auth      smtp.Auth
	user      string
	host      string
	port      int
	jwtSecret string
}

func NewEmailService() (*EmailService, error) {
	smtpCfg, err := config.LoadSMTPConfig()
	if err != nil {
		return nil, err
	}

	cfg := utils.ToNonPointer(smtpCfg)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	log.Printf("Loaded SMTP Config: %v\n", string(data))

	return &EmailService{
		auth:      smtp.PlainAuth("", cfg.SMTP.User, cfg.SMTP.Password, cfg.SMTP.Host),
		user:      cfg.SMTP.User,
		host:      cfg.SMTP.Host,
		port:      cfg.SMTP.Port,
		jwtSecret: cfg.SMTP.JWTSecret,
	}, nil
}

func (s *EmailService) Send(to, subject, body string) error {
	message := fmt.Appendf(nil,
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n", s.user, to, subject, body)

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", s.host, s.port),
		s.auth, s.user, []string{to}, message)

	if err != nil {
		return err
	}

	log.Println("Mail sent")
	return nil
}

func (s *EmailService) sendSimpleDemoEmail() error {
	subject := "Sample Test Email"
	to := "tqthost@gmail.com"
	body := `
	<html>
		<body style="font-family: Arial, sans-serif">
			<h2 style="color: #4caf50">Test</h2>
			<p>Sample test email</p>
			<ul>
				<li>Support HTML</li>
				<li>Support CSS</li>
				<li>Link: <a href="https://golang.org">Go Documentation</a></li>
			</ul>
		</body>
	</html>
	`
	return s.Send(to, subject, body)
}

func (s *EmailService) GenerateResetToken(isJwt bool, userId int) (string, error) {
	if isJwt {
		claims := jwt.MapClaims{
			"user_id": userId,
			"exp":     time.Now().Add(15 * time.Minute).Unix(), // Expired in 15mins
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		return token.SignedString([]byte(s.jwtSecret))
	}

	// Should load from config, in this example can be use the hard code string
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, 32)
	for i := range token {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		token[i] = charset[num.Int64()]
	}
	return string(token), nil
}

func (s *EmailService) Generate2FACode() (string, error) {
	num, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", num.Int64()), nil
}

func (s *EmailService) Send2FACode(to, code string) error {
	subject := "Request 2FA"
	templatePath, err := filepath.Abs("email/2fa.html")
	if err != nil {
		return err
	}

	htmlTemplate, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	var body bytes.Buffer // buffer store the render template result
	// use strings.Builder or execute template
	err = htmlTemplate.Execute(&body, struct{ Code string }{Code: code})
	if err != nil {
		return err
	}

	return s.Send(to, subject, body.String())
}

func (s *EmailService) SendPasswordReset(to, name string) error {
	userId := 100
	subject := "Request Reset Password"
	token, err := s.GenerateResetToken(true, userId)
	if err != nil {
		return err
	}

	resetLink := fmt.Sprintf("http://localhost:8080/reset-password?token=%s", token)

	templatePath, err := filepath.Abs("email/reset-password.html")
	if err != nil {
		return err
	}

	htmlTemplate, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	var body strings.Builder
	err = htmlTemplate.Execute(&body, struct{ ResetLink string }{ResetLink: resetLink})
	if err != nil {
		return err
	}

	return s.Send(to, subject, body.String())
}

func RunSimpleExample(skip bool) {
	if skip {
		return
	}

	s, err := NewEmailService()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.sendSimpleDemoEmail(); err != nil {
		log.Fatal(err)
	}
}
