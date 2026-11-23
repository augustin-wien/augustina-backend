package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/stretchr/testify/require"
)

func TestMailQueueJobs(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	ctx := context.Background()

	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	sent := 0
	mailer.Send = func(*mailer.EmailRequest) (bool, error) {
		sent++
		return true, nil
	}

	due, err := database.Db.EntClient.MailTask.Create().
		SetKind(database.MailKindWelcome).SetRecipients("due@example.com").SetSubject("s").SetBody("b").
		SetNextAttemptAt(time.Now().Add(-time.Minute)).Save(ctx)
	require.NoError(t, err)
	old, err := database.Db.EntClient.MailTask.Create().
		SetKind(database.MailKindWelcome).SetRecipients("old@example.com").SetSubject("s").SetBody("personal link").
		SetStatus(database.MailTaskSent).SetSentAt(time.Now().Add(-mailContentRetention - time.Hour)).Save(ctx)
	require.NoError(t, err)

	processMailQueue()
	require.Equal(t, 1, sent)
	task, err := database.Db.GetMailTask(due.ID)
	require.NoError(t, err)
	require.Equal(t, database.MailTaskSent, task.Status)

	purgeSentMailContent()
	task, err = database.Db.GetMailTask(old.ID)
	require.NoError(t, err)
	require.Empty(t, task.Body)
	require.Equal(t, database.MailTaskSent, task.Status, "the delivery record stays")

	// Nothing left to purge
	purgeSentMailContent()
}

func TestOrderReconciliationCanBeDisabled(t *testing.T) {
	orig := config.Config.OrderReconcileIntervalMinutes
	t.Cleanup(func() { config.Config.OrderReconcileIntervalMinutes = orig })
	config.Config.OrderReconcileIntervalMinutes = 0
	StartOrderReconciliation()
}
