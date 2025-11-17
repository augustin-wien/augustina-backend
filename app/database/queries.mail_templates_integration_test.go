//go:build integration
// +build integration

package database

import (
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

// TestMain not defined here; this is an integration test and will initialize DB itself.
func TestMailTemplateDBFlow(t *testing.T) {
	// ensure running from repo root like other tests
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)

	name := "it_test_template"
	subject := "Hello {{.Name}}"
	body := "<p>Dear {{.Name}}</p>"

	// create
	err = Db.CreateOrUpdateMailTemplate(name, subject, body)
	require.NoError(t, err)

	// fetch
	mt, err := Db.GetMailTemplateByName(name)
	require.NoError(t, err)
	require.Equal(t, name, mt.Name)
	require.Equal(t, subject, mt.Subject)

	// build email request from template and render
	req, err := Db.BuildEmailRequestFromTemplate(name, []string{"to@example.test"}, map[string]string{"Name": "Bob"})
	require.NoError(t, err)
	require.Contains(t, req.Subject(), "Bob")
	require.Contains(t, req.Body(), "Bob")
}
