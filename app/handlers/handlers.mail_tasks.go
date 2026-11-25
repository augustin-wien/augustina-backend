package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// mailTaskFilterFromQuery reads the list filter from the query string
func mailTaskFilterFromQuery(r *http.Request) (database.MailTaskFilter, error) {
	q := r.URL.Query()
	f := database.MailTaskFilter{
		Status: q.Get("status"),
		Kind:   q.Get("kind"),
		Search: q.Get("search"),
	}
	switch f.Status {
	case "", database.MailTaskPending, database.MailTaskSending, database.MailTaskSent, database.MailTaskFailed:
	default:
		return f, errors.New("invalid status")
	}
	for name, dst := range map[string]*int{"order_id": &f.OrderID, "item_id": &f.ItemID, "limit": &f.Limit, "offset": &f.Offset} {
		if v := q.Get(name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return f, errors.New("invalid " + name)
			}
			*dst = n
		}
	}
	return f, nil
}

// ListMailTasks godoc
//
//	@Summary		List outgoing mails
//	@Description	Lists the mails the backend sent or is sending, newest first, with the number of mails per delivery state. Bodies are left out.
//	@Tags			Mail tasks
//	@Produce		json
//	@Param			status		query		string	false	"pending, sending, sent or failed"
//	@Param			kind		query		string	false	"Kind of mail, e.g. online_issue"
//	@Param			item_id		query		int		false	"Item the mail belongs to"
//	@Param			order_id	query		int		false	"Order the mail belongs to"
//	@Param			search		query		string	false	"Part of a recipient address or the subject"
//	@Param			limit		query		int		false	"Page size (default 100, max 500)"
//	@Param			offset		query		int		false	"Offset"
//	@Success		200			{object}	database.MailTaskList
//	@Security		KeycloakAuth
//	@Router			/mail-tasks/ [get]
func ListMailTasks(w http.ResponseWriter, r *http.Request) {
	f, err := mailTaskFilterFromQuery(r)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	list, err := database.Db.ListMailTasks(f)
	if err != nil {
		log.Error("ListMailTasks: ", err)
		utils.ErrorJSON(w, errors.New("listing mails failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, list)
}

// GetMailTask godoc
//
//	@Summary		Get an outgoing mail
//	@Description	Returns an outgoing mail including its body. The body of a mail sent more than 30 days ago is empty.
//	@Tags			Mail tasks
//	@Produce		json
//	@Param			id	path		int	true	"Mail task ID"
//	@Success		200	{object}	database.MailTask
//	@Security		KeycloakAuth
//	@Router			/mail-tasks/{id}/ [get]
func GetMailTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid id"), http.StatusBadRequest)
		return
	}
	task, err := database.Db.GetMailTask(id)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, errors.New("mail not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("GetMailTask: ", err)
		utils.ErrorJSON(w, errors.New("loading the mail failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, task)
}

// RetryMailTaskResponse names the task that was queued
type RetryMailTaskResponse struct {
	ID int
}

// RetryMailTask godoc
//
//	@Summary		Send an outgoing mail again
//	@Description	Queues a pending or failed mail right away with a fresh set of attempts. For a sent mail a copy is queued, so the earlier delivery stays on record; this is not possible once its body was removed after 30 days.
//	@Tags			Mail tasks
//	@Produce		json
//	@Param			id	path		int	true	"Mail task ID"
//	@Success		200	{object}	RetryMailTaskResponse
//	@Security		KeycloakAuth
//	@Router			/mail-tasks/{id}/retry/ [post]
func RetryMailTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid id"), http.StatusBadRequest)
		return
	}
	queued, err := database.Db.RetryMailTask(id)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, errors.New("mail not found"), http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrMailTaskNotRetryable) {
		utils.ErrorJSON(w, err, http.StatusConflict)
		return
	}
	if err != nil {
		log.Error("RetryMailTask: ", err)
		utils.ErrorJSON(w, errors.New("queueing the mail failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, RetryMailTaskResponse{ID: queued})
}

// RetryFailedMailTasksResponse tells how many mails were queued again
type RetryFailedMailTasksResponse struct {
	Queued int
}

// RetryFailedMailTasks godoc
//
//	@Summary		Send all failed mails again
//	@Description	Queues every failed mail matching the filter again, e.g. all failed mails of an online issue.
//	@Tags			Mail tasks
//	@Produce		json
//	@Param			kind		query		string	false	"Kind of mail, e.g. online_issue"
//	@Param			item_id		query		int		false	"Item the mail belongs to"
//	@Param			order_id	query		int		false	"Order the mail belongs to"
//	@Param			search		query		string	false	"Part of a recipient address or the subject"
//	@Success		200			{object}	RetryFailedMailTasksResponse
//	@Security		KeycloakAuth
//	@Router			/mail-tasks/retry-failed/ [post]
func RetryFailedMailTasks(w http.ResponseWriter, r *http.Request) {
	f, err := mailTaskFilterFromQuery(r)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	n, err := database.Db.RetryFailedMailTasks(f)
	if err != nil {
		log.Error("RetryFailedMailTasks: ", err)
		utils.ErrorJSON(w, errors.New("queueing the mails failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, RetryFailedMailTasksResponse{Queued: n})
}

// GetOnlineIssueMailStatus godoc
//
//	@Summary		Mail delivery state of an online issue
//	@Description	Sums up the mails sent to the abonnents for an online issue: how many are pending, sent or failed, when the issue was first queued and when the last mail went out.
//	@Tags			Items
//	@Produce		json
//	@Param			id	path		int	true	"Item ID"
//	@Success		200	{object}	database.MailDeliveryStatus
//	@Security		KeycloakAuth
//	@Router			/items/{id}/notify-abonements/ [get]
func GetOnlineIssueMailStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid item id"), http.StatusBadRequest)
		return
	}
	status, err := database.Db.GetMailDeliveryStatusForItem(id, database.MailKindOnlineIssue)
	if err != nil {
		log.Error("GetOnlineIssueMailStatus: ", err)
		utils.ErrorJSON(w, errors.New("loading the mail status failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, status)
}
