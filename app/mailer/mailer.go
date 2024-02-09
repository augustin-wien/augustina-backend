package mailer

import (
	"augustin/config"
	"augustin/utils"
	"bytes"

	"net/smtp"
	"text/template"
)

var log = utils.GetLogger()

var auth smtp.Auth

func Init() {
	log.Infoln("Initializing mailer")
	auth = smtp.PlainAuth("", config.Config.SMTPUsername, config.Config.SMTPPassword, config.Config.SMTPServer)
}

// Request struct
type EmailRequest struct {
	to      []string
	subject string
	body    string
}

func NewRequestFromTemplate(to []string, subject, templateFileName string, data interface{}) (*EmailRequest, error) {
	r := NewRequest(to, subject, "")
	err := r.ParseTemplate(templateFileName, data)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func NewRequest(to []string, subject, body string) *EmailRequest {
	return &EmailRequest{
		to:      to,
		subject: subject,
		body:    body,
	}
}

func (r *EmailRequest) SendEmail() (bool, error) {
	log.Info("Sending email to ", r.to, " with subject ", r.subject)
	from := "From: " + config.Config.SMTPSenderAddress + "\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: " + r.subject + "!\n"
	msg := []byte(from + subject + mime + "\n" + r.body)
	addr := config.Config.SMTPServer + ":" + config.Config.SMTPPort

	if err := smtp.SendMail(addr, auth, config.Config.SMTPUsername, r.to, msg); err != nil {
		log.Error("Error sending email ", err)
		return false, err
	}
	return true, nil
}

func (r *EmailRequest) ParseTemplate(templateFileName string, data interface{}) error {
	t, err := template.ParseFiles("./templates/" + templateFileName)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return err
	}
	r.body = buf.String()
	return nil
}
