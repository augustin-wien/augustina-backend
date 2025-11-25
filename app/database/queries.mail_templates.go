package database

import (
	"bytes"
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/augustin-wien/augustina-backend/mailer"
)

// MailTemplate represents a mail template stored in the database
type MailTemplate struct {
	ID        int
	Name      string
	Subject   string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetMailTemplateByName fetches a mail template by its name
func (db *Database) GetMailTemplateByName(name string) (MailTemplate, error) {
	var mt MailTemplate
	row := db.Dbpool.QueryRow(context.Background(), `SELECT id, name, subject, body, created_at, updated_at FROM mail_templates WHERE name=$1`, name)
	err := row.Scan(&mt.ID, &mt.Name, &mt.Subject, &mt.Body, &mt.CreatedAt, &mt.UpdatedAt)
	if err != nil {
		return mt, err
	}
	return mt, nil
}

// ListMailTemplates returns all templates
func (db *Database) ListMailTemplates() ([]MailTemplate, error) {
	rows, err := db.Dbpool.Query(context.Background(), `SELECT id, name, subject, body, created_at, updated_at FROM mail_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MailTemplate
	for rows.Next() {
		var mt MailTemplate
		if err := rows.Scan(&mt.ID, &mt.Name, &mt.Subject, &mt.Body, &mt.CreatedAt, &mt.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, mt)
	}
	return out, nil
}

// CreateOrUpdateMailTemplate inserts or updates a template by name
func (db *Database) CreateOrUpdateMailTemplate(name, subject, body string) error {
	_, err := db.Dbpool.Exec(context.Background(), `
        INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
        VALUES ($1, $2, $3, now(), now())
        ON CONFLICT (name) DO UPDATE SET subject = EXCLUDED.subject, body = EXCLUDED.body, updated_at = now();
    `, name, subject, body)
	return err
}

// DeleteMailTemplate deletes a template by name
func (db *Database) DeleteMailTemplate(name string) error {
	_, err := db.Dbpool.Exec(context.Background(), `DELETE FROM mail_templates WHERE name=$1`, name)
	return err
}

// BuildEmailRequestFromTemplate builds a mailer.EmailRequest by loading a template from the DB
func (db *Database) BuildEmailRequestFromTemplate(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
	mt, err := db.GetMailTemplateByName(name)
	if err != nil {
		return nil, err
	}
	subj := mt.Subject
	if strings.Contains(subj, "{{") {
		t, err := template.New("subject").Parse(mt.Subject)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, err
		}
		subj = buf.String()
	}
	r := mailer.NewRequest(to, subj, "")
	if err := r.ParseTemplateFromString(mt.Body, data); err != nil {
		return nil, err
	}
	return r, nil
}

// BuildEmailRequestFromTemplate is a package-level function variable that by default
// forwards to the method on the global Db. Tests can override this variable to
// provide a stubbed implementation.
var BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
	return Db.BuildEmailRequestFromTemplate(name, to, data)
}
