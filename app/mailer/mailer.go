package mailer

import (
	"augustin/config"
	"augustin/utils"
	"bytes"
	"crypto/tls"
	"errors"

	"net/smtp"
	"text/template"
)

var log = utils.GetLogger()

var auth smtp.Auth

func Init() {
	log.Infoln("Initializing mailer")
	host := config.Config.SMTPServer + ":" + config.Config.SMTPPort
	if !config.Config.SMTPSsl {
		host = config.Config.SMTPServer
	}
	auth = smtp.PlainAuth("", config.Config.SMTPUsername, config.Config.SMTPPassword, host)
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

	from := "From: " + config.Config.SMTPSenderAddress + "\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: " + r.subject + "!\n"
	msg := []byte(from + subject + mime + "\n" + r.body)

	if !config.Config.SMTPSsl {
		log.Info("Sending email to ", r.to, " with subject ", r.subject)
		addr := config.Config.SMTPServer + ":" + config.Config.SMTPPort
		if err := smtp.SendMail(addr, auth, config.Config.SMTPUsername, r.to, msg); err != nil {
			log.Error("SendEmail: noSSL: Error sending email ", err)
			return false, err
		}
		return true, nil
	} else {
		log.Info("Sending ssl email to ", r.to, " with subject ", r.subject)

		// TLS config
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         config.Config.SMTPServer,
		}
		host := config.Config.SMTPServer + ":" + config.Config.SMTPPort

		// Here is the key, you need to call tls.Dial instead of smtp.Dial
		// for smtp servers running on 465 that require an ssl connection
		// from the very beginning (no starttls)
		conn, err := tls.Dial("tcp", host, tlsconfig)
		if err != nil {
			log.Error("SendEmail: failed tls dial", err)
			return false, err
		}

		c, err := smtp.NewClient(conn, host)
		if err != nil {
			log.Error("SendMail: Ssl: create smtp client", err)
			return false, err

		}

		// Auth
		if err = c.Auth(auth); err != nil {
			log.Error("SendMail: Ssl:", err)
			return false, err
		}

		// To && From
		if err = c.Mail(config.Config.SMTPSenderAddress); err != nil {
			log.Error("SendMail: Ssl: create mail: ", err)
			return false, err
		}

		if err = c.Rcpt(r.to[0]); err != nil {
			log.Error("SendMail: Ssl: set recipient:", err)
			return false, err
		}

		// Data
		w, err := c.Data()
		if err != nil {
			log.Error("SendMail: Ssl: data", err)
			return false, err
		}

		_, err = w.Write([]byte(msg))
		if err != nil {
			log.Error("SendMail: Ssl: send message", err)
			return false, err
		}

		err = w.Close()
		if err != nil {
			log.Error("SendMail: Ssl: close connection", err)
			return false, err
		}

		c.Quit()
		return true, nil
	}

}

func (r *EmailRequest) ParseTemplate(templateFileName string, data interface{}) error {
	// Parse the template
	if templateFileName == "" {
		return errors.New("template file name is empty")
	}
	path := "./templates/" + templateFileName

	if !utils.FileExists(path) {
		return errors.New("template file does not exist")
	}

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
