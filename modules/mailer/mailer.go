package mailer

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func SendHTML(to, subject, body string) error {
	provider := strings.ToUpper(strings.TrimSpace(os.Getenv("EMAIL_PROVIDER")))
	if provider != "" && provider != "SMTP" {
		return fmt.Errorf("email provider no soportado: %s", provider)
	}

	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	from := firstNonEmpty(os.Getenv("EMAIL_FROM"), os.Getenv("SMTP_FROM"), os.Getenv("SMTP_USER"))
	user := os.Getenv("SMTP_USER")
	pass := firstNonEmpty(os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_PASS"))

	if host == "" || port == "" || from == "" {
		return fmt.Errorf("smtp no configurado")
	}

	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	msg := buildMessage(from, to, subject, body)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func buildMessage(from, to, subject, body string) []byte {
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}
	return []byte(strings.Join(headers, "\r\n"))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
