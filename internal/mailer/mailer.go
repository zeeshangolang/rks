package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var TemplateFs embed.FS

//contains a mail.Dialer instances to connect to stmp server
//and ther sender information to send email

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

func (m Mailer) Send(recipitent, TemplateFile string, data any) error {

	templ, err := template.New("email").ParseFS(TemplateFs, "templates/"+TemplateFile)
	if err != nil {
		return err
	}
	subject := new(bytes.Buffer)
	err = templ.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = templ.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	html := new(bytes.Buffer)
	err = templ.ExecuteTemplate(html, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipitent)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", html.String())

	err = m.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}
	return nil
}
