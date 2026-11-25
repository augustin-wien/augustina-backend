package handlers

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestMailTaskEndpoints(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	itemID, err := database.Db.CreateItem(database.Item{Name: "Mail Task Issue", Description: "Test item", Price: 300, Type: "online_issue"})
	require.NoError(t, err)

	ctx := t.Context()
	failed, err := database.Db.EntClient.MailTask.Create().
		SetKind(database.MailKindOnlineIssue).
		SetRecipients("failed@example.com").
		SetSubject("Neue Ausgabe").
		SetBody("<p>Body</p>").
		SetStatus(database.MailTaskFailed).
		SetAttempts(5).
		SetItemID(itemID).
		Save(ctx)
	require.NoError(t, err)
	sending, err := database.Db.EntClient.MailTask.Create().
		SetKind(database.MailKindWelcome).
		SetRecipients("sending@example.com").
		SetSubject("Willkommen").
		SetBody("<p>Hi</p>").
		SetStatus(database.MailTaskSending).
		SetNextAttemptAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/mail-tasks/?status=failed&kind=online_issue&item_id="+strconv.Itoa(itemID)+"&limit=10&offset=0", nil, 200, adminUserToken)
	var list database.MailTaskList
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &list))
	require.Len(t, list.Tasks, 1)
	require.Equal(t, failed.ID, list.Tasks[0].ID)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/mail-tasks/"+strconv.Itoa(failed.ID)+"/", nil, 200, adminUserToken)
	var task database.MailTask
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &task))
	require.Equal(t, "<p>Body</p>", task.Body)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/items/"+strconv.Itoa(itemID)+"/notify-abonements/", nil, 200, adminUserToken)
	var status database.MailDeliveryStatus
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &status))

	res = utils.TestRequestWithAuth(t, r, "POST", "/api/mail-tasks/retry-failed/?item_id="+strconv.Itoa(itemID), nil, 200, adminUserToken)
	var retried RetryFailedMailTasksResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &retried))
	require.Equal(t, 1, retried.Queued)

	// A mail that is being sent right now cannot be queued again
	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-tasks/"+strconv.Itoa(sending.ID)+"/retry/", nil, 409, adminUserToken)
	// Wait until the background delivery started by retry-failed is finished
	require.Eventually(t, func() bool {
		task, err := database.Db.GetMailTask(failed.ID)
		return err == nil && task.Status != database.MailTaskPending && task.Status != database.MailTaskSending
	}, 5*time.Second, 20*time.Millisecond)
	res = utils.TestRequestWithAuth(t, r, "POST", "/api/mail-tasks/"+strconv.Itoa(failed.ID)+"/retry/", nil, 200, adminUserToken)
	var queued RetryMailTaskResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &queued))
	// The mail went out, so a copy is queued
	require.NotEqual(t, failed.ID, queued.ID)

	for path, code := range map[string]int{
		"/api/mail-tasks/?status=lost":      400,
		"/api/mail-tasks/?limit=many":       400,
		"/api/mail-tasks/abc/":              400,
		"/api/mail-tasks/999999/":           404,
		"/api/items/abc/notify-abonements/": 400,
	} {
		utils.TestRequestWithAuth(t, r, "GET", path, nil, code, adminUserToken)
	}
	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-tasks/abc/retry/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-tasks/999999/retry/", nil, 404, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-tasks/retry-failed/?order_id=x", nil, 400, adminUserToken)
}
