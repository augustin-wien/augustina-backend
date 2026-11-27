package database

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/stretchr/testify/require"
)

// stubKeycloakActionMail replaces the Keycloak call for the test and records the sent mails
func stubKeycloakActionMail(t *testing.T, result error) *[]keycloak.ActionMail {
	t.Helper()
	var mu sync.Mutex
	sent := []keycloak.ActionMail{}
	orig := sendKeycloakActionMail
	t.Cleanup(func() { sendKeycloakActionMail = orig })
	sendKeycloakActionMail = func(m keycloak.ActionMail) error {
		mu.Lock()
		defer mu.Unlock()
		if result != nil {
			return result
		}
		sent = append(sent, m)
		return nil
	}
	return &sent
}

var testActionMail = keycloak.ActionMail{
	UserID:      "kc-user-1",
	Recipient:   "reader@example.com",
	RedirectURI: "https://zeitung.example.com",
	Action:      "UPDATE_PASSWORD",
}

func TestSendKeycloakActionMail_RecordsSentMail(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	sent := stubKeycloakActionMail(t, nil)

	require.NoError(t, Db.SendKeycloakActionMail(testActionMail))
	require.Equal(t, []keycloak.ActionMail{testActionMail}, *sent)

	list, err := Db.ListMailTasks(MailTaskFilter{Kind: MailKindKeycloakUpdatePassword})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	task := list.Tasks[0]
	require.Equal(t, MailTaskSent, task.Status)
	require.Equal(t, []string{"reader@example.com"}, task.Recipients)

	full, err := Db.GetMailTask(task.ID)
	require.NoError(t, err)
	require.Empty(t, full.Body, "the stored Keycloak call is not shown as mail content")
}

func TestSendKeycloakActionMail_KindPerAction(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubKeycloakActionMail(t, nil)

	verify := testActionMail
	verify.Action = "VERIFY_EMAIL"
	require.NoError(t, Db.SendKeycloakActionMail(verify))

	list, err := Db.ListMailTasks(MailTaskFilter{Kind: MailKindKeycloakVerifyEmail})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
}

func TestSendKeycloakActionMail_TransientFailureIsRetriedLater(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubKeycloakActionMail(t, &gocloak.APIError{Code: http.StatusInternalServerError, Message: "Failed to send execute actions email"})

	require.Error(t, Db.SendKeycloakActionMail(testActionMail))

	list, err := Db.ListMailTasks(MailTaskFilter{})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	require.Equal(t, MailTaskPending, list.Tasks[0].Status)
	require.Contains(t, list.Tasks[0].LastError, "Failed to send")
}

func TestSendKeycloakActionMail_ClientErrorFailsRightAway(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubKeycloakActionMail(t, &gocloak.APIError{Code: http.StatusNotFound, Message: "User not found"})

	require.Error(t, Db.SendKeycloakActionMail(testActionMail))

	list, err := Db.ListMailTasks(MailTaskFilter{Status: MailTaskFailed})
	require.NoError(t, err)
	require.Len(t, list.Tasks, 1)
	require.Equal(t, 1, list.Tasks[0].Attempts)
}

func TestRetryMailTask_KeycloakMailIsSentAgain(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	stubKeycloakActionMail(t, errors.New("connection refused"))
	require.Error(t, Db.SendKeycloakActionMail(testActionMail))
	list, err := Db.ListMailTasks(MailTaskFilter{})
	require.NoError(t, err)
	id := list.Tasks[0].ID

	sent := stubKeycloakActionMail(t, nil)
	_, err = Db.RetryMailTask(id)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		task, err := Db.GetMailTask(id)
		return err == nil && task.Status == MailTaskSent
	}, 5*time.Second, 20*time.Millisecond)

	// Old sent Keycloak mails keep what is needed to send them again
	n, err := Db.PurgeSentMailContent(time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Zero(t, n)

	copied, err := Db.RetryMailTask(id)
	require.NoError(t, err)
	require.NotEqual(t, id, copied)
	require.Eventually(t, func() bool {
		task, err := Db.GetMailTask(copied)
		return err == nil && task.Status == MailTaskSent
	}, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, []keycloak.ActionMail{testActionMail, testActionMail}, *sent)
}
