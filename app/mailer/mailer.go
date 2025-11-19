package mailer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"net/smtp"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"
)

var log = utils.GetLogger()

var auth smtp.Auth

// encodeRFC2047 encodes a header value (Subject) if it contains non-ASCII
// characters using the 'encoded-word' syntax from RFC 2047 (base64, UTF-8).
func encodeRFC2047(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
		}
	}
	return s
}

func Init() {
	log.Infoln("Initializing mailer")
	// smtp.PlainAuth requires the server name (without port)
	host := config.Config.SMTPServer
	auth = smtp.PlainAuth("", config.Config.SMTPUsername, config.Config.SMTPPassword, host)

}

// loginAuth implements the AUTH LOGIN SASL mechanism.
type loginAuth struct {
	username string
	password string
	step     int
}

// LoginAuth returns an smtp.Auth that implements the LOGIN mechanism.
func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username: username, password: password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// LOGIN has no initial response
	a.step = 0
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	// Server will prompt for username then password. We ignore the actual prompt content and return base64 values.
	if a.step == 0 {
		a.step++
		return []byte(a.username), nil
	}
	if a.step == 1 {
		a.step++
		return []byte(a.password), nil
	}
	return nil, nil
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

// SetSubject sets the email subject (exported helper for other packages)
func (r *EmailRequest) SetSubject(s string) {
	r.subject = s
}

// Subject returns the email subject
func (r *EmailRequest) Subject() string {
	return r.subject
}

// Body returns the email body (HTML)
func (r *EmailRequest) Body() string {
	return r.body
}

// BuildMessage builds the raw email message bytes (multipart/alternative)
// and returns the message and the boundary used. This is separated so tests
// can inspect the generated message without sending it.
func (r *EmailRequest) BuildMessage() ([]byte, string, error) {
	from := config.Config.SMTPSenderAddress
	toHeader := strings.Join(r.to, ", ")
	dateHeader := time.Now().UTC().Format(time.RFC1123Z)
	domain := config.Config.SMTPServer
	if parts := strings.Split(from, "@"); len(parts) == 2 {
		domain = parts[1]
	}
	msgID := fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), time.Now().Unix(), domain)

	// create a plain-text alternative by stripping HTML tags (simple fallback)
	stripTags := regexp.MustCompile("<[^>]*>")
	plain := stripTags.ReplaceAllString(r.body, "")
	if strings.TrimSpace(plain) == "" {
		plain = "(no plain-text body)"
	}

	boundary := fmt.Sprintf("Boundary_%d", time.Now().UnixNano())
	var b bytes.Buffer

	// headers
	encodedSubject := encodeRFC2047(r.subject)
	// Build From header with optional display name
	fromName := config.Config.SMTPSenderName
	var fromHeader string
	if strings.TrimSpace(fromName) != "" {
		fromHeader = fmt.Sprintf("%s <%s>", encodeRFC2047(fromName), from)
	} else {
		fromHeader = from
	}
	b.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	b.WriteString(fmt.Sprintf("To: %s\r\n", toHeader))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	b.WriteString(fmt.Sprintf("Date: %s\r\n", dateHeader))
	b.WriteString(fmt.Sprintf("Message-ID: %s\r\n", msgID))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	b.WriteString("\r\n")

	// plain part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(plain)); err != nil {
		return nil, "", fmt.Errorf("failed to encode plain text: %w", err)
	}
	if err := qp.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close plain text encoder: %w", err)
	}
	b.WriteString("\r\n")

	// html part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	qp2 := quotedprintable.NewWriter(&b)
	if _, err := qp2.Write([]byte(r.body)); err != nil {
		return nil, "", fmt.Errorf("failed to encode HTML body: %w", err)
	}
	if err := qp2.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close HTML body encoder: %w", err)
	}
	b.WriteString("\r\n")

	// end
	b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return b.Bytes(), boundary, nil
}

func (r *EmailRequest) SendEmail() (bool, error) {
	// Build the message bytes
	msg, _, err := r.BuildMessage()
	if err != nil {
		return false, err
	}

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
		log.Info("SendEmail: Sending (STARTTLS) email to ", r.to, " with subject ", r.subject)
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
		if ok, mech := c.Extension("AUTH"); ok {
			log.Infof("SendEmail: SMTP server advertised AUTH mechanisms: %s", mech)
			m := strings.ToUpper(mech)
			var chosen smtp.Auth
			if strings.Contains(m, "PLAIN") {
				chosen = auth
			} else if strings.Contains(m, "LOGIN") {
				chosen = LoginAuth(config.Config.SMTPUsername, config.Config.SMTPPassword)
			} else if strings.Contains(m, "XOAUTH2") {
				log.Infof("SendEmail: SMTP server requires XOAUTH2; no XOAUTH2 support configured in this build")
				return false, errors.New("server requires XOAUTH2 authentication")
			} else {
				// fallback to client configured auth
				chosen = auth
			}
			if chosen != nil {
				if err = c.Auth(chosen); err != nil {
					log.Error("SendEmail: auth failed", err)
					return false, err
				}
			}
		} else {
			if auth != nil {
				if err = c.Auth(auth); err != nil {
					log.Error("SendEmail: auth failed", err)
					return false, err
				}
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
		log.Info("Email sent successfully via STARTTLS")
		return true, nil
	}

	// Implicit TLS path (SMTPS, typically port 465)
	log.Info("SendEmail: Sending SSL (implicit TLS) email to ", r.to, " with subject ", r.subject)
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

	if ok, mech := c.Extension("AUTH"); ok {
		log.Infof("SendEmail: SMTP server advertised AUTH mechanisms: %s", mech)
		m := strings.ToUpper(mech)
		var chosen smtp.Auth
		if strings.Contains(m, "PLAIN") {
			chosen = auth
		} else if strings.Contains(m, "LOGIN") {
			chosen = LoginAuth(config.Config.SMTPUsername, config.Config.SMTPPassword)
		} else if strings.Contains(m, "XOAUTH2") {
			log.Infof("SendEmail: SMTP server requires XOAUTH2; no XOAUTH2 support configured in this build")
			return false, errors.New("server requires XOAUTH2 authentication")
		} else {
			chosen = auth
		}
		if chosen != nil {
			if err = c.Auth(chosen); err != nil {
				log.Error("SendEmail: auth failed", err)
				return false, err
			}
		}
	} else {
		if auth != nil {
			if err = c.Auth(auth); err != nil {
				log.Error("SendEmail: auth failed", err)
				return false, err
			}
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
	log.Info("SendEmail: Email sent successfully via implicit TLS")
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

// ParseTemplateFromString parses a template from the provided string content
func (r *EmailRequest) ParseTemplateFromString(templateContent string, data interface{}) error {
	if templateContent == "" {
		return errors.New("template content is empty")
	}
	t, err := template.New("mail").Parse(templateContent)
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
