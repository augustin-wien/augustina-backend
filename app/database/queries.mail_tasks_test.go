package database

import (
	"context"
	"errors"
	"net/textproto"
	"sync"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/stretchr/testify/require"
)

// stubMailSend replaces mailer.Send for the test and records the sent mails
func stubMailSend(t *testing.T, result error) *[]*mailer.EmailRequest {
	t.Helper()
	var mu sync.Mutex
	sent := []*mailer.EmailRequest{}
	orig := mailer.Send
	t.Cleanup(func() { mailer.Send = orig })
	mailer.Send = func(m *mailer.EmailRequest) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		if result != nil {
			return false, result
		}
		sent = append(sent, m)
		return true, nil
	}
	return &sent
}

func newTestMail(t *testing.T, to string) *mailer.EmailRequest {
	t.Helper()
	m, err := mailer.NewRequest([]string{to}, "Betreff", "<p>Hallo</p>")
	require.NoError(t, err)
	return m
}

func TestSendMailNow_RecordsSentMail(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	sent := stubMailSend(t, nil)

	m := newTestMail(t, "reader@example.com")
	require.NoError(t, m.AddAttachment("gutschein.pdf", "application/pdf", []byte("%PDF-1.4")))
	require.NoError(t, Db.SendMailNow(m, MailMeta{Kind: MailKindWelcome, OrderID: 7}))

	require.Len(t, *sent, 1)
	require.Equal(t, []string{"reader@example.com"}, (*sent)[0].To())
	require.Equal(t, "Betreff", (*sent)[0].Subject())
	require.Len(t, (*sent)[0].Attachments(), 1)
	require.Equal(t, []byte("%PDF-1.4"), (*sent)[0].Attachments()[0].Data)

	list, err := Db.ListMailTasks(MailTaskFilter{OrderID: 7})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	task := list.Tasks[0]
	require.Equal(t, MailTaskSent, task.Status)
	require.Equal(t, 1, task.Attempts)
	require.True(t, task.SentAt.Valid)
	require.True(t, task.HasAttachments)
	require.Empty(t, task.Body, "the list leaves bodies out")
	require.Equal(t, 1, list.Counts[MailTaskSent])

	full, err := Db.GetMailTask(task.ID)
	require.NoError(t, err)
	require.Equal(t, "<p>Hallo</p>", full.Body)
}

func TestDeliverMailTask_TransientFailureIsRetriedLater(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, errors.New("dial tcp: connection refused"))

	err := Db.SendMailNow(newTestMail(t, "reader@example.com"), MailMeta{Kind: MailKindOnlineIssue, ItemID: 3})
	require.Error(t, err)

	list, err := Db.ListMailTasks(MailTaskFilter{ItemID: 3})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	task := list.Tasks[0]
	require.Equal(t, MailTaskPending, task.Status)
	require.Equal(t, 1, task.Attempts)
	require.Contains(t, task.LastError, "connection refused")
	require.True(t, task.NextAttemptAt.After(time.Now()), "the next attempt waits")

	n, err := Db.ProcessDueMailTasks(10)
	require.NoError(t, err)
	require.Zero(t, n, "a retry is not due yet")
}

func TestDeliverMailTask_PermanentFailureFailsRightAway(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, &textproto.Error{Code: 550, Msg: "mailbox unavailable"})

	require.Error(t, Db.SendMailNow(newTestMail(t, "gone@example.com"), MailMeta{Kind: MailKindWelcome}))

	list, err := Db.ListMailTasks(MailTaskFilter{Status: MailTaskFailed})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	require.Equal(t, 1, list.Tasks[0].Attempts)
}

func TestDeliverMailTask_FailsAfterLastRetry(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, errors.New("timeout"))

	require.Error(t, Db.SendMailNow(newTestMail(t, "reader@example.com"), MailMeta{Kind: MailKindWelcome}))
	list, err := Db.ListMailTasks(MailTaskFilter{})
	require.NoError(t, err)
	id := list.Tasks[0].ID

	for i := 1; i <= len(mailRetryDelays); i++ {
		require.NoError(t, Db.EntClient.MailTask.UpdateOneID(id).SetNextAttemptAt(time.Now().Add(-time.Second)).Exec(context.Background()))
		n, err := Db.ProcessDueMailTasks(10)
		require.NoError(t, err)
		require.Equal(t, 1, n)
	}

	task, err := Db.GetMailTask(id)
	require.NoError(t, err)
	require.Equal(t, MailTaskFailed, task.Status)
	require.Equal(t, len(mailRetryDelays)+1, task.Attempts)
}

func TestProcessDueMailTasks_PicksUpStaleSendingTask(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	sent := stubMailSend(t, nil)

	id, err := Db.createMailTask(newTestMail(t, "reader@example.com"), MailMeta{Kind: MailKindWelcome})
	require.NoError(t, err)
	_, err = Db.EntClient.MailTask.UpdateOneID(id).
		SetStatus(MailTaskSending).
		SetUpdatedAt(time.Now().Add(-time.Hour)).
		Save(context.Background())
	require.NoError(t, err)

	n, err := Db.ProcessDueMailTasks(10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Len(t, *sent, 1)
}

func TestRetryMailTask(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, &textproto.Error{Code: 550, Msg: "rejected"})
	require.Error(t, Db.SendMailNow(newTestMail(t, "reader@example.com"), MailMeta{Kind: MailKindOnlineIssue, ItemID: 9}))
	list, err := Db.ListMailTasks(MailTaskFilter{})
	require.NoError(t, err)
	failedID := list.Tasks[0].ID

	sent := stubMailSend(t, nil)
	queued, err := Db.RetryMailTask(failedID)
	require.NoError(t, err)
	require.Equal(t, failedID, queued, "a failed task is sent again as it is")
	require.Eventually(t, func() bool {
		task, err := Db.GetMailTask(failedID)
		return err == nil && task.Status == MailTaskSent
	}, 5*time.Second, 20*time.Millisecond)
	require.Len(t, *sent, 1)

	copied, err := Db.RetryMailTask(failedID)
	require.NoError(t, err)
	require.NotEqual(t, failedID, copied, "a sent task is copied, its record stays")
	require.Eventually(t, func() bool {
		task, err := Db.GetMailTask(copied)
		return err == nil && task.Status == MailTaskSent
	}, 5*time.Second, 20*time.Millisecond)
	task, err := Db.GetMailTask(copied)
	require.NoError(t, err)
	require.Equal(t, int64(9), task.ItemID.Int64)
}

func TestRetryMailTask_SentMailWithoutContentCannotBeSentAgain(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, nil)
	require.NoError(t, Db.SendMailNow(newTestMail(t, "reader@example.com"), MailMeta{Kind: MailKindWelcome}))

	n, err := Db.PurgeSentMailContent(time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, n)

	list, err := Db.ListMailTasks(MailTaskFilter{})
	require.NoError(t, err)
	_, err = Db.RetryMailTask(list.Tasks[0].ID)
	require.ErrorIs(t, err, ErrMailTaskNotRetryable)
}

func TestRetryFailedMailTasks_OnlyMatchingItem(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, &textproto.Error{Code: 550, Msg: "rejected"})
	require.Error(t, Db.SendMailNow(newTestMail(t, "a@example.com"), MailMeta{Kind: MailKindOnlineIssue, ItemID: 1}))
	require.Error(t, Db.SendMailNow(newTestMail(t, "b@example.com"), MailMeta{Kind: MailKindOnlineIssue, ItemID: 2}))

	sent := stubMailSend(t, nil)
	n, err := Db.RetryFailedMailTasks(MailTaskFilter{ItemID: 1})
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Eventually(t, func() bool { return len(*sent) == 1 }, 5*time.Second, 20*time.Millisecond)

	list, err := Db.ListMailTasks(MailTaskFilter{ItemID: 2})
	require.NoError(t, err)
	require.Equal(t, MailTaskFailed, list.Tasks[0].Status)
}

func TestMailDeliveryStateForItem(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubMailSend(t, nil)
	require.NoError(t, Db.SendMailNow(newTestMail(t, "A@example.com"), MailMeta{Kind: MailKindOnlineIssue, ItemID: 5}))
	stubMailSend(t, errors.New("timeout"))
	require.Error(t, Db.SendMailNow(newTestMail(t, "b@example.com"), MailMeta{Kind: MailKindOnlineIssue, ItemID: 5}))
	require.Error(t, Db.SendMailNow(newTestMail(t, "c@example.com"), MailMeta{Kind: MailKindWelcome, ItemID: 5}))

	recipients, err := Db.MailRecipientsForItem(5, MailKindOnlineIssue)
	require.NoError(t, err)
	require.Equal(t, map[string]bool{"a@example.com": true, "b@example.com": true}, recipients)

	status, err := Db.GetMailDeliveryStatusForItem(5, MailKindOnlineIssue)
	require.NoError(t, err)
	require.Equal(t, 2, status.Total)
	require.Equal(t, 1, status.Counts[MailTaskSent])
	require.Equal(t, 1, status.Counts[MailTaskPending])
	require.True(t, status.LastSentAt.Valid)
	require.True(t, status.FirstQueuedAt.Valid)
}
