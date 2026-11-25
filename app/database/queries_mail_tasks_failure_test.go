package database

import (
	"context"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/stretchr/testify/require"
)

func TestMailTaskQueriesReturnDatabaseErrors(t *testing.T) {
	db := brokenDatabase(t)
	m, err := mailer.NewRequest([]string{"a@example.com"}, "s", "b")
	require.NoError(t, err)
	meta := MailMeta{Kind: MailKindWelcome, OrderID: 1, ItemID: 1}

	_, err = db.QueueMail(m, meta)
	require.Error(t, err)
	ok, err := db.QueueMailFunc(meta)(m)
	require.Error(t, err)
	require.False(t, ok)
	require.Error(t, db.SendMailNow(m, meta))
	require.Error(t, db.DeliverMailTask(1))
	_, err = db.ProcessDueMailTasks(10)
	require.Error(t, err)
	_, err = db.PurgeSentMailContent(time.Now())
	require.Error(t, err)
	_, err = db.ListMailTasks(MailTaskFilter{})
	require.Error(t, err)
	_, err = db.GetMailTask(1)
	require.Error(t, err)
	_, err = db.RetryMailTask(1)
	require.Error(t, err)
	_, err = db.RetryFailedMailTasks(MailTaskFilter{})
	require.Error(t, err)
	_, err = db.MailRecipientsForItem(1, MailKindOnlineIssue)
	require.Error(t, err)
	_, err = db.GetMailDeliveryStatusForItem(1, MailKindOnlineIssue)
	require.Error(t, err)

	// Mails without content or recipients are not queued
	_, err = db.QueueMail(nil, meta)
	require.Error(t, err)
	empty, err := mailer.NewRequest(nil, "s", "b")
	require.NoError(t, err)
	_, err = db.QueueMail(empty, meta)
	require.Error(t, err)
}

func TestDeliverMailTaskSendFailures(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	ctx := context.Background()

	// A send that reports no success is retried later
	mailer.Send = func(*mailer.EmailRequest) (bool, error) { return false, nil }
	m, err := mailer.NewRequest([]string{"a@example.com"}, "s", "b")
	require.NoError(t, err)
	require.NoError(t, m.AddAttachment("a.pdf", "application/pdf", []byte("%PDF")))
	require.Error(t, Db.SendMailNow(m, MailMeta{Kind: MailKindWelcome}))
	list, err := Db.ListMailTasks(MailTaskFilter{Status: MailTaskPending})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	require.True(t, list.Tasks[0].HasAttachments)

	// A task that is not pending is left alone
	e, err := Db.EntClient.MailTask.Create().SetKind(MailKindWelcome).SetRecipients("a@example.com").SetSubject("s").SetBody("b").SetStatus(MailTaskSent).Save(ctx)
	require.NoError(t, err)
	require.NoError(t, Db.DeliverMailTask(e.ID))
}
