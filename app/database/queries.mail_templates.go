package database

import (
	"bytes"
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/ent/mailtemplate"
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

	t, err := db.EntClient.MailTemplate.Query().Where(mailtemplate.Name(name)).First(context.Background())
	if err != nil {
		return mt, err
	}

	mt.ID = t.ID
	mt.Name = t.Name
	mt.Subject = t.Subject
	mt.Body = t.Body
	mt.CreatedAt = t.CreatedAt
	mt.UpdatedAt = t.UpdatedAt
	return mt, nil
}

// ListMailTemplates returns all templates
func (db *Database) ListMailTemplates() ([]MailTemplate, error) {
	list, err := db.EntClient.MailTemplate.Query().Order(ent.Asc(mailtemplate.FieldName)).All(context.Background())
	if err != nil {
		return nil, err
	}

	var out []MailTemplate
	for _, t := range list {
		out = append(out, MailTemplate{
			ID:        t.ID,
			Name:      t.Name,
			Subject:   t.Subject,
			Body:      t.Body,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}
	return out, nil
}

// CreateOrUpdateMailTemplate inserts or updates a template by name
func (db *Database) CreateOrUpdateMailTemplate(name, subject, body string) error {
	// Using Upsert via OnConflict is dialect specific in Ent, or we can use the Upsert feature if generated.
	// Since standard Ent doesn't generate Upsert (OnConflict) methods without correct feature flags,
	// and enabling them requires changing generate.go which I didn't see flags for,
	// I will use a simple check-and-update pattern or simple recreate.
	// Actually "entgo.io/ent/dialect/sql" allows it but cleaner to use Ent semantics.
	// Let's try to find, if exists update, else create.

	ctx := context.Background()
	exists, err := db.EntClient.MailTemplate.Query().Where(mailtemplate.Name(name)).Exist(ctx)
	if err != nil {
		return err
	}

	if exists {
		_, err = db.EntClient.MailTemplate.Update().
			Where(mailtemplate.Name(name)).
			SetSubject(subject).
			SetBody(body).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = db.EntClient.MailTemplate.Create().
			SetName(name).
			SetSubject(subject).
			SetBody(body).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	return err
}

// DeleteMailTemplate deletes a template by name
func (db *Database) DeleteMailTemplate(name string) error {
	_, err := db.EntClient.MailTemplate.Delete().Where(mailtemplate.Name(name)).Exec(context.Background())
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
	r, err := mailer.NewRequest(to, subj, "")
	if err != nil {
		return nil, err
	}
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
