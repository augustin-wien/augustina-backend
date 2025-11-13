package mailer

import (
	"bytes"
	"crypto/tls"
	"errors"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"

	"net/smtp"
	"text/template"
)

var log = utils.GetLogger()

var auth smtp.Auth

func Init() {
	log.Infoln("Initializing mailer")
	// smtp.PlainAuth requires the server name (without port)
	host := config.Config.SMTPServer
	auth = smtp.PlainAuth("", config.Config.SMTPUsername, config.Config.SMTPPassword, host)

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

	addr := config.Config.SMTPServer + ":" + config.Config.SMTPPort

	// Decide whether to use STARTTLS or implicit TLS.
	useStartTLS := !config.Config.SMTPSsl
	if config.Config.SMTPSsl && config.Config.SMTPPort == "587" {
		// explicit override: SMTPSsl set but port indicates STARTTLS (Office365)
		useStartTLS = true
		log.Infoln("SMTPSsl=true with port 587 detected — using STARTTLS (Office365) instead of implicit TLS")
	}

	if useStartTLS {
		// Try to use STARTTLS if the server supports it (Office365 requires STARTTLS on port 587)
		log.Info("Sending (STARTTLS) email to ", r.to, " with subject ", r.subject)
		c, err := smtp.Dial(addr)
		if err != nil {
			log.Error("SendEmail: dial error", err)
			return false, err
		}
		defer c.Close()

		// Upgrade to TLS if supported
		if ok, _ := c.Extension("STARTTLS"); ok {
			tlsconfig := &tls.Config{
				ServerName:         config.Config.SMTPServer,
				InsecureSkipVerify: config.Config.SMTPInsecureSkipVerify,
			}
			if err = c.StartTLS(tlsconfig); err != nil {
				log.Error("SendEmail: STARTTLS failed", err)
				return false, err
			}
		}

		// Auth
		// Log server advertised AUTH mechanisms (helpful for diagnosing unsupported auth errors)
		if ok, mech := c.Extension("AUTH"); ok {
			log.Infof("SMTP server advertised AUTH mechanisms: %s", mech)
		}
		if auth != nil {
			if err = c.Auth(auth); err != nil {
				log.Error("SendEmail: auth failed", err)
				return false, err
			}
		}

		// From
		if err = c.Mail(config.Config.SMTPSenderAddress); err != nil {
			log.Error("SendEmail: set mail failed", err)
			return false, err
		}

		// To
		for _, rcpt := range r.to {
			if err = c.Rcpt(rcpt); err != nil {
				log.Error("SendEmail: rcpt failed", err)
				return false, err
			}
		}

		// Data
		w, err := c.Data()
		if err != nil {
			log.Error("SendEmail: data failed", err)
			return false, err
		}
		if _, err = w.Write([]byte(msg)); err != nil {
			log.Error("SendEmail: write failed", err)
			return false, err
		}
		if err = w.Close(); err != nil {
			log.Error("SendEmail: close failed", err)
			return false, err
		}
		if err = c.Quit(); err != nil {
			log.Error("SendEmail: quit failed", err)
			return false, err
		}
		return true, nil
	}

	// Implicit TLS path (SMTPS, typically port 465)
	log.Info("Sending SSL (implicit TLS) email to ", r.to, " with subject ", r.subject)
	tlsconfig := &tls.Config{
		ServerName:         config.Config.SMTPServer,
		InsecureSkipVerify: config.Config.SMTPInsecureSkipVerify,
	}
	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		log.Error("SendEmail: TLS dial failed", err)
		return false, err
	}
	c, err := smtp.NewClient(conn, config.Config.SMTPServer)
	if err != nil {
		log.Error("SendEmail: create smtp client failed", err)
		return false, err
	}
	defer c.Close()

	// Log server advertised AUTH mechanisms (helpful for diagnosing unsupported auth errors)
	if ok, mech := c.Extension("AUTH"); ok {
		log.Infof("SMTP server advertised AUTH mechanisms: %s", mech)
	}

	if auth != nil {
		if err = c.Auth(auth); err != nil {
			log.Error("SendEmail: auth failed", err)
			return false, err
		}
	}
	if err = c.Mail(config.Config.SMTPSenderAddress); err != nil {
		log.Error("SendEmail: set mail failed", err)
		return false, err
	}
	for _, rcpt := range r.to {
		if err = c.Rcpt(rcpt); err != nil {
			log.Error("SendEmail: rcpt failed", err)
			return false, err
		}
	}
	w, err := c.Data()
	if err != nil {
		log.Error("SendEmail: data failed", err)
		return false, err
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		log.Error("SendEmail: write failed", err)
		return false, err
	}
	if err = w.Close(); err != nil {
		log.Error("SendEmail: close failed", err)
		return false, err
	}
	if err = c.Quit(); err != nil {
		log.Error("SendEmail: quit failed", err)
		return false, err
	}
	return true, nil

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
