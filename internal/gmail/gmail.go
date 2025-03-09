package gmail

import (
	"errors"
	"github.com/fentezi/export-word/config"
	"gopkg.in/gomail.v2"
)

type Message struct {
	From    string
	To      []string
	Subject string
	Body    string
	File    string
}

type Gmail struct {
	dial *gomail.Dialer
}

func New(gm config.Gmail) *Gmail {
	dial := gomail.NewDialer("smtp.gmail.com", 587, gm.Email, gm.Password)
	return &Gmail{
		dial: dial,
	}
}

func (g *Gmail) SendMessage(message Message) error {
	if message.From == "" {
		return errors.New("from field is empty")
	}
	if len(message.To) == 0 {
		return errors.New("to field is empty")
	}
	if message.Subject == "" {
		return errors.New("subject field is empty")
	}
	if message.Body == "" {
		return errors.New("body field is empty")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", message.From)
	msg.SetHeader("To", message.To...)

	msg.SetHeader("Subject", message.Subject)
	msg.SetBody("text/html", message.Body)

	if message.File != "" {
		msg.Attach(message.File)
	}

	return g.dial.DialAndSend(msg)
}
