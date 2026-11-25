package mailer

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"
)

// encodeRFC2047 encodes a header value (Subject) if it contains non-ASCII
// characters or control characters using the 'encoded-word' syntax from RFC 2047 (base64, UTF-8).
func encodeRFC2047(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 || s[i] == '\r' || s[i] == '\n' {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
		}
	}
	return s
}

// Request struct
type EmailRequest struct {
	to          []string
	subject     string
	body        string
	attachments []Attachment
}

// Attachment is a file sent along with an email
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// AddAttachment attaches a file to the email
func (r *EmailRequest) AddAttachment(filename, contentType string, data []byte) error {
	if strings.ContainsAny(filename, "\r\n\"") || strings.ContainsAny(contentType, "\r\n") {
		return errors.New("attachment name or type contains invalid characters")
	}
	r.attachments = append(r.attachments, Attachment{Filename: filename, ContentType: contentType, Data: data})
	return nil
}

// Attachments returns the files attached to the email
func (r *EmailRequest) Attachments() []Attachment {
	return r.attachments
}

func NewRequestFromTemplate(to []string, subject, templateFileName string, data interface{}) (*EmailRequest, error) {
	r, err := NewRequest(to, subject, "")
	if err != nil {
		return nil, err
	}
	err = r.ParseTemplate(templateFileName, data)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func sanitizeAndValidateRecipients(recipients []string) ([]string, error) {
	sanitized := make([]string, 0, len(recipients))
	for _, r := range recipients {
		s := strings.TrimSpace(r)
		if s == "" {
			continue
		}
		if strings.ContainsAny(s, "\r\n") {
			return nil, errors.New("recipient address contains invalid characters")
		}
		sanitized = append(sanitized, s)
	}
	return sanitized, nil
}

func NewRequest(to []string, subject, body string) (*EmailRequest, error) {
	cleanTo, err := sanitizeAndValidateRecipients(to)
	if err != nil {
		return nil, err
	}
	// Also validate subject for injection
	if strings.ContainsAny(subject, "\r\n") {
		return nil, errors.New("subject contains invalid characters")
	}

	return &EmailRequest{
		to:      cleanTo,
		subject: subject,
		body:    body,
	}, nil
}

// SetSubject sets the email subject (exported helper for other packages)
func (r *EmailRequest) SetSubject(s string) {
	r.subject = s
}

// Subject returns the email subject
func (r *EmailRequest) Subject() string {
	return r.subject
}

// To returns the email recipients
func (r *EmailRequest) To() []string {
	return r.to
}

// Body returns the email body (HTML)
func (r *EmailRequest) Body() string {
	return r.body
}

var (
	plainLinkRe = regexp.MustCompile(`(?is)<a\s[^>]*?href\s*=\s*["']([^"']*)["'][^>]*>(.*?)</a>`)
	plainTagRe  = regexp.MustCompile("<[^>]*>")
)

// htmlToPlain builds the plain-text alternative by stripping HTML tags (simple
// fallback). Links are rewritten to "text (url)" first, otherwise the URL - the
// one thing most of our mails exist to deliver - is lost from the text part.
func htmlToPlain(body string) string {
	plain := plainLinkRe.ReplaceAllStringFunc(body, func(a string) string {
		m := plainLinkRe.FindStringSubmatch(a)
		href := strings.TrimSpace(m[1])
		text := strings.TrimSpace(plainTagRe.ReplaceAllString(m[2], ""))
		if href == "" || href == text {
			return text
		}
		if text == "" {
			return href
		}
		return text + " (" + href + ")"
	})
	return plainTagRe.ReplaceAllString(plain, "")
}

// BuildMessage builds the raw email message bytes (multipart/alternative)
// and returns the message and the boundary used. This is separated so tests
// can inspect the generated message without sending it.
func (r *EmailRequest) BuildMessage() ([]byte, string, error) {
	from := config.Config.SMTPSenderAddress
	toHeader := strings.Join(r.to, ", ")
	// Defense in depth: catch any injection attempts in To header
	if strings.ContainsAny(toHeader, "\r\n") {
		return nil, "", errors.New("header injection attempt in To")
	}

	dateHeader := time.Now().UTC().Format(time.RFC1123Z)
	domain := config.Config.SMTPServer
	if parts := strings.Split(from, "@"); len(parts) == 2 {
		domain = parts[1]
	}
	// Sanitize domain to be safe (though it comes from config)
	domain = strings.ReplaceAll(domain, "\r", "")
	domain = strings.ReplaceAll(domain, "\n", "")

	msgID := fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), time.Now().Unix(), domain)

	plain := htmlToPlain(r.body)
	if strings.TrimSpace(plain) == "" {
		plain = "(no plain-text body)"
	}

	boundary := fmt.Sprintf("Boundary_%d", time.Now().UnixNano())
	var b bytes.Buffer

	// headers
	encodedSubject := encodeRFC2047(r.subject)
	// Build From header with optional display name
	fromName := config.Config.MailSenderName
	if strings.TrimSpace(fromName) == "" {
		fromName = config.Config.SMTPSenderName
	}
	var fromHeader string
	if strings.TrimSpace(fromName) != "" {
		fromHeader = fmt.Sprintf("%s <%s>", encodeRFC2047(fromName), from)
	} else {
		fromHeader = from
	}
	fmt.Fprintf(&b, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&b, "To: %s\r\n", toHeader)
	fmt.Fprintf(&b, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&b, "Date: %s\r\n", dateHeader)
	fmt.Fprintf(&b, "Message-ID: %s\r\n", msgID)
	b.WriteString("MIME-Version: 1.0\r\n")
	mixedBoundary := "Mixed" + boundary
	if len(r.attachments) > 0 {
		fmt.Fprintf(&b, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n", mixedBoundary)
		b.WriteString("\r\n")
		fmt.Fprintf(&b, "--%s\r\n", mixedBoundary)
	}
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
	b.WriteString("\r\n")

	// plain part
	fmt.Fprintf(&b, "--%s\r\n", boundary)
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
	fmt.Fprintf(&b, "--%s\r\n", boundary)
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
	fmt.Fprintf(&b, "--%s--\r\n", boundary)

	if len(r.attachments) > 0 {
		for _, a := range r.attachments {
			fmt.Fprintf(&b, "--%s\r\n", mixedBoundary)
			fmt.Fprintf(&b, "Content-Type: %s; name=\"%s\"\r\n", a.ContentType, a.Filename)
			b.WriteString("Content-Transfer-Encoding: base64\r\n")
			fmt.Fprintf(&b, "Content-Disposition: attachment; filename=\"%s\"\r\n", a.Filename)
			b.WriteString("\r\n")
			encoded := base64.StdEncoding.EncodeToString(a.Data)
			for len(encoded) > 76 {
				b.WriteString(encoded[:76] + "\r\n")
				encoded = encoded[76:]
			}
			b.WriteString(encoded + "\r\n")
		}
		fmt.Fprintf(&b, "--%s--\r\n", mixedBoundary)
	}

	return b.Bytes(), boundary, nil
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
	// Normalize/augment data so templates can use either uppercase or
	// lowercase keys (URL/url and EMAIL/email). If data is nil, create a
	// default map with both forms populated.
	if data == nil {
		var email string
		if len(r.to) > 0 {
			email = r.to[0]
		}
		data = map[string]interface{}{
			"URL":   config.Config.OnlinePaperUrl,
			"url":   config.Config.OnlinePaperUrl,
			"EMAIL": email,
			"email": email,
		}
	} else {
		// If caller passed a map, ensure both key casings exist.
		if m, ok := data.(map[string]interface{}); ok {
			// URL/url
			if v, exists := m["URL"]; exists {
				if _, ok := m["url"]; !ok {
					m["url"] = v
				}
			} else if v, exists := m["url"]; exists {
				if _, ok := m["URL"]; !ok {
					m["URL"] = v
				}
			} else {
				// neither set; fill from config
				m["URL"] = config.Config.OnlinePaperUrl
				m["url"] = config.Config.OnlinePaperUrl
			}
			// EMAIL/email
			if v, exists := m["EMAIL"]; exists {
				if _, ok := m["email"]; !ok {
					m["email"] = v
				}
			} else if v, exists := m["email"]; exists {
				if _, ok := m["EMAIL"]; !ok {
					m["EMAIL"] = v
				}
			} else {
				var email string
				if len(r.to) > 0 {
					email = r.to[0]
				}
				m["EMAIL"] = email
				m["email"] = email
			}
			data = m
		}
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
