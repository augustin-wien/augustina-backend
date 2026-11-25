package mailer

import (
	"crypto/tls"
	"errors"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"
)

var log = utils.GetLogger()

var auth smtp.Auth

// Send is a package-level function that forwards to EmailRequest.SendEmail.
// Tests can override this to avoid network calls.
var Send = func(r *EmailRequest) (bool, error) {
	return r.SendEmail()
}

// retryDelays are the waits before each further attempt of SendWithRetry.
var retryDelays = []time.Duration{15 * time.Second, time.Minute, 5 * time.Minute}

// isTransientSMTPError reports whether the SMTP server answered with a 4xx
// reply, i.e. asked to try again later.
func isTransientSMTPError(err error) bool {
	var protoErr *textproto.Error
	return errors.As(err, &protoErr) && protoErr.Code >= 400 && protoErr.Code < 500
}

// IsPermanentSMTPError reports whether the SMTP server rejected the mail with a
// permanent (5xx) reply, so sending it again will not help.
func IsPermanentSMTPError(err error) bool {
	var protoErr *textproto.Error
	return errors.As(err, &protoErr) && protoErr.Code >= 500 && protoErr.Code < 600
}

// SendWithRetry sends like Send and tries again when the SMTP server answers
// with a transient (4xx) error. It blocks for the waits in between, so use it
// from background goroutines only.
func SendWithRetry(r *EmailRequest) (bool, error) {
	success, err := Send(r)
	for _, delay := range retryDelays {
		if !isTransientSMTPError(err) {
			break
		}
		log.Warn("SendWithRetry: transient SMTP error, retrying in ", delay, ": ", err)
		time.Sleep(delay)
		success, err = Send(r)
	}
	return success, err
}

func Init() {
	log.Infoln("Initializing mailer")
	// smtp.PlainAuth requires the server name (without port)
	host := config.Config.SMTPServer
	auth = smtp.PlainAuth("", config.Config.SMTPUsername, config.Config.SMTPPassword, host)

	// Enforce safe TLS behaviour:
	// - In production (Development == false) we fail-fast if InsecureSkipVerify is enabled.
	// - In development we require an explicit opt-in via ALLOW_INSECURE_SMTP=true if InsecureSkipVerify is enabled.
	if config.Config.SMTPInsecureSkipVerify {
		if !config.Config.Development {
			log.Fatal("SMTP insecure skip verify is true in production; refusing to start")
		}
		// In development require explicit opt-in
		if !config.Config.AllowInsecureSMTP {
			log.Fatalf("SMTP insecure skip verify is enabled; set ALLOW_INSECURE_SMTP=true to allow this in development")
		}
	}
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
		defer func() { _ = c.Close() }()

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

		// Auth — skip entirely when no credentials are configured (e.g. Mailpit)
		if config.Config.SMTPUsername != "" {
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
			} else if auth != nil {
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
	defer func() { _ = c.Close() }()

	// Auth — skip entirely when no credentials are configured (e.g. Mailpit)
	if config.Config.SMTPUsername != "" {
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
		} else if auth != nil {
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
