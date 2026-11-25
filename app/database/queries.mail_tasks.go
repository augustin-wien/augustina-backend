package database

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/ent/mailtask"
	"github.com/augustin-wien/augustina-backend/ent/predicate"
	"github.com/augustin-wien/augustina-backend/mailer"
	"gopkg.in/guregu/null.v4"
)

// Delivery states of a mail task
const (
	MailTaskPending = "pending"
	MailTaskSending = "sending"
	MailTaskSent    = "sent"
	MailTaskFailed  = "failed"
)

// Kinds of mail tasks
const (
	MailKindOnlineIssue           = "online_issue"
	MailKindWelcome               = "welcome"
	MailKindDigitalLicence        = "digital_licence"
	MailKindPDFLicence            = "pdf_licence"
	MailKindVoucher               = "voucher"
	MailKindAbonementConfirmation = "abonement_confirmation"
	MailKindAccountingReport      = "accounting_report"
	MailKindFeedback              = "feedback"
	MailKindTemplateTest          = "template_test"
)

// mailRetryDelays are the waits before each further attempt of a failed mail. Once
// they are used up the task is marked failed and waits for a manual retry.
var mailRetryDelays = []time.Duration{
	time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	6 * time.Hour,
}

// staleSendingAfter is how long a task may stay in "sending" before it is assumed
// that the process died while sending and the task is picked up again
const staleSendingAfter = 10 * time.Minute

// mailSendSlots limits how many mails are sent at the same time, so queueing a
// mail for every abonnent does not open hundreds of SMTP connections at once
var mailSendSlots = make(chan struct{}, 2)

var (
	// ErrMailTaskNotRetryable is returned when a task cannot be sent again
	ErrMailTaskNotRetryable = errors.New("mail task cannot be sent again")
)

// MailMeta describes what a queued mail belongs to
type MailMeta struct {
	Kind    string
	OrderID int
	ItemID  int
}

// MailTask is an outgoing mail and its delivery state
type MailTask struct {
	ID             int
	Kind           string
	Recipients     []string
	Subject        string
	Body           string `json:",omitempty"`
	HasAttachments bool
	Status         string
	Attempts       int
	LastError      string
	NextAttemptAt  time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	SentAt         null.Time `swaggertype:"string"`
	OrderID        null.Int  `swaggertype:"integer"`
	ItemID         null.Int  `swaggertype:"integer"`
}

// MailTaskFilter narrows down ListMailTasks
type MailTaskFilter struct {
	Status  string
	Kind    string
	OrderID int
	ItemID  int
	Search  string
	Limit   int
	Offset  int
}

// MailTaskList is a page of mail tasks with the number of tasks per status
type MailTaskList struct {
	Tasks  []MailTask
	Total  int
	Counts map[string]int
}

func convertMailTask(e *ent.MailTask, withBody bool) MailTask {
	t := MailTask{
		ID:             e.ID,
		Kind:           e.Kind,
		Recipients:     splitRecipients(e.Recipients),
		Subject:        e.Subject,
		HasAttachments: e.Attachments != nil && len(*e.Attachments) > 0,
		Status:         e.Status,
		Attempts:       e.Attempts,
		NextAttemptAt:  e.NextAttemptAt,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
	if withBody {
		t.Body = e.Body
	}
	if e.LastError != nil {
		t.LastError = *e.LastError
	}
	if e.SentAt != nil {
		t.SentAt = null.TimeFrom(*e.SentAt)
	}
	if e.OrderID != nil {
		t.OrderID = null.IntFrom(int64(*e.OrderID))
	}
	if e.ItemID != nil {
		t.ItemID = null.IntFrom(int64(*e.ItemID))
	}
	return t
}

func splitRecipients(s string) []string {
	out := []string{}
	for _, r := range strings.Split(s, ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// createMailTask stores the mail as a pending task
func (db *Database) createMailTask(m *mailer.EmailRequest, meta MailMeta) (int, error) {
	if m == nil {
		return 0, errors.New("no mail to queue")
	}
	if len(m.To()) == 0 {
		return 0, errors.New("mail has no recipients")
	}
	create := db.EntClient.MailTask.Create().
		SetKind(meta.Kind).
		SetRecipients(strings.Join(m.To(), ",")).
		SetSubject(m.Subject()).
		SetBody(m.Body())
	if attachments := m.Attachments(); len(attachments) > 0 {
		data, err := json.Marshal(attachments)
		if err != nil {
			return 0, err
		}
		create = create.SetAttachments(data)
	}
	if meta.OrderID != 0 {
		create = create.SetOrderID(meta.OrderID)
	}
	if meta.ItemID != 0 {
		create = create.SetItemID(meta.ItemID)
	}
	e, err := create.Save(context.Background())
	if err != nil {
		log.Error("createMailTask: ", err)
		return 0, err
	}
	return e.ID, nil
}

// QueueMail stores the mail as a task and sends it in the background. A failed
// attempt is retried by the mail queue job.
func (db *Database) QueueMail(m *mailer.EmailRequest, meta MailMeta) (int, error) {
	id, err := db.createMailTask(m, meta)
	if err != nil {
		return 0, err
	}
	go func() { _ = db.DeliverMailTask(id) }()
	return id, nil
}

// QueueMailFunc returns a send function, for code that takes one, that queues
// the mail with the given meta data instead of sending it directly
func (db *Database) QueueMailFunc(meta MailMeta) func(*mailer.EmailRequest) (bool, error) {
	return func(m *mailer.EmailRequest) (bool, error) {
		if _, err := db.QueueMail(m, meta); err != nil {
			return false, err
		}
		return true, nil
	}
}

// SendMailNow stores the mail as a task and sends it right away. On failure the
// error is returned and the task stays queued for retries.
func (db *Database) SendMailNow(m *mailer.EmailRequest, meta MailMeta) error {
	id, err := db.createMailTask(m, meta)
	if err != nil {
		return err
	}
	return db.DeliverMailTask(id)
}

// DeliverMailTask sends a pending task. It does nothing if the task is not
// pending, e.g. because another goroutine is already sending it.
func (db *Database) DeliverMailTask(id int) error {
	// Claimed only once a slot is free, a waiting task must not look stuck in "sending"
	mailSendSlots <- struct{}{}
	defer func() { <-mailSendSlots }()

	ctx := context.Background()
	claimed, err := db.EntClient.MailTask.Update().
		Where(mailtask.ID(id), mailtask.StatusEQ(MailTaskPending)).
		SetStatus(MailTaskSending).
		Save(ctx)
	if err != nil {
		log.Error("DeliverMailTask: claiming task failed: ", id, err)
		return err
	}
	if claimed == 0 {
		return nil
	}

	e, err := db.EntClient.MailTask.Get(ctx, id)
	if err != nil {
		log.Error("DeliverMailTask: loading task failed: ", id, err)
		return err
	}

	sendErr := sendMailTask(e)
	attempts := e.Attempts + 1
	update := db.EntClient.MailTask.UpdateOneID(id).SetAttempts(attempts)
	if sendErr == nil {
		update = update.SetStatus(MailTaskSent).SetSentAt(time.Now()).ClearLastError()
	} else if mailer.IsPermanentSMTPError(sendErr) || attempts > len(mailRetryDelays) {
		update = update.SetStatus(MailTaskFailed).SetLastError(sendErr.Error())
	} else {
		update = update.SetStatus(MailTaskPending).
			SetLastError(sendErr.Error()).
			SetNextAttemptAt(time.Now().Add(mailRetryDelays[attempts-1]))
	}
	if err := update.Exec(ctx); err != nil {
		log.Error("DeliverMailTask: saving delivery state failed: ", id, err)
	}

	if sendErr != nil {
		if attempts > len(mailRetryDelays) || mailer.IsPermanentSMTPError(sendErr) {
			log.Errorw("DeliverMailTask: mail could not be sent", "mail_task_id", id, "kind", e.Kind, "attempts", attempts, "error", sendErr)
		} else {
			log.Warnw("DeliverMailTask: sending mail failed, will retry", "mail_task_id", id, "kind", e.Kind, "attempts", attempts, "error", sendErr)
		}
	}
	return sendErr
}

// sendMailTask rebuilds the mail of a task and sends it
func sendMailTask(e *ent.MailTask) error {
	m, err := mailer.NewRequest(splitRecipients(e.Recipients), e.Subject, e.Body)
	if err != nil {
		return err
	}
	if e.Attachments != nil && len(*e.Attachments) > 0 {
		var attachments []mailer.Attachment
		if err := json.Unmarshal(*e.Attachments, &attachments); err != nil {
			return err
		}
		for _, a := range attachments {
			if err := m.AddAttachment(a.Filename, a.ContentType, a.Data); err != nil {
				return err
			}
		}
	}
	success, err := mailer.Send(m)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("mail was not sent")
	}
	return nil
}

// ProcessDueMailTasks sends up to limit pending tasks whose next attempt is due and
// picks up tasks left in "sending" by a process that stopped while sending.
// Returns the number of tasks it tried to send.
func (db *Database) ProcessDueMailTasks(limit int) (int, error) {
	ctx := context.Background()
	if _, err := db.EntClient.MailTask.Update().
		Where(mailtask.StatusEQ(MailTaskSending), mailtask.UpdatedAtLT(time.Now().Add(-staleSendingAfter))).
		SetStatus(MailTaskPending).
		Save(ctx); err != nil {
		log.Error("ProcessDueMailTasks: resetting stale tasks failed: ", err)
	}

	ids, err := db.EntClient.MailTask.Query().
		Where(mailtask.StatusEQ(MailTaskPending), mailtask.NextAttemptAtLTE(time.Now())).
		Order(ent.Asc(mailtask.FieldNextAttemptAt), ent.Asc(mailtask.FieldID)).
		Limit(limit).
		IDs(ctx)
	if err != nil {
		log.Error("ProcessDueMailTasks: loading due tasks failed: ", err)
		return 0, err
	}
	for _, id := range ids {
		_ = db.DeliverMailTask(id)
	}
	return len(ids), nil
}

// PurgeSentMailContent empties body and attachments of mails sent before the
// cutoff. The delivery record itself is kept.
func (db *Database) PurgeSentMailContent(before time.Time) (int, error) {
	return db.EntClient.MailTask.Update().
		Where(mailtask.StatusEQ(MailTaskSent), mailtask.SentAtLT(before), mailtask.BodyNEQ("")).
		SetBody("").
		ClearAttachments().
		Save(context.Background())
}

func mailTaskPredicates(f MailTaskFilter, withStatus bool) []predicate.MailTask {
	var ps []predicate.MailTask
	if withStatus && f.Status != "" {
		ps = append(ps, mailtask.StatusEQ(f.Status))
	}
	if f.Kind != "" {
		ps = append(ps, mailtask.KindEQ(f.Kind))
	}
	if f.OrderID != 0 {
		ps = append(ps, mailtask.OrderID(f.OrderID))
	}
	if f.ItemID != 0 {
		ps = append(ps, mailtask.ItemID(f.ItemID))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		ps = append(ps, mailtask.Or(mailtask.RecipientsContainsFold(s), mailtask.SubjectContainsFold(s)))
	}
	return ps
}

// ListMailTasks returns the newest mail tasks matching the filter, without their bodies
func (db *Database) ListMailTasks(f MailTaskFilter) (MailTaskList, error) {
	ctx := context.Background()
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	list := MailTaskList{Tasks: []MailTask{}, Counts: map[string]int{
		MailTaskPending: 0, MailTaskSending: 0, MailTaskSent: 0, MailTaskFailed: 0,
	}}

	query := db.EntClient.MailTask.Query().Where(mailTaskPredicates(f, true)...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return list, err
	}
	list.Total = total

	rows, err := query.
		Order(ent.Desc(mailtask.FieldCreatedAt), ent.Desc(mailtask.FieldID)).
		Limit(f.Limit).
		Offset(f.Offset).
		All(ctx)
	if err != nil {
		return list, err
	}
	for _, e := range rows {
		list.Tasks = append(list.Tasks, convertMailTask(e, false))
	}

	var counts []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	err = db.EntClient.MailTask.Query().
		Where(mailTaskPredicates(f, false)...).
		GroupBy(mailtask.FieldStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &counts)
	if err != nil {
		return list, err
	}
	for _, c := range counts {
		list.Counts[c.Status] = c.Count
	}
	return list, nil
}

// GetMailTask returns a mail task including its body
func (db *Database) GetMailTask(id int) (MailTask, error) {
	e, err := db.EntClient.MailTask.Get(context.Background(), id)
	if err != nil {
		return MailTask{}, err
	}
	return convertMailTask(e, true), nil
}

// RetryMailTask sends a mail task again. A pending or failed task is queued
// right away with a fresh set of attempts; for a sent task a copy is queued, so
// the record of the earlier delivery is kept. Returns the id of the queued task.
func (db *Database) RetryMailTask(id int) (int, error) {
	ctx := context.Background()
	e, err := db.EntClient.MailTask.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	switch e.Status {
	case MailTaskPending, MailTaskFailed:
		err = db.EntClient.MailTask.UpdateOneID(id).
			SetStatus(MailTaskPending).
			SetAttempts(0).
			SetNextAttemptAt(time.Now()).
			Exec(ctx)
		if err != nil {
			return 0, err
		}
	case MailTaskSent:
		if e.Body == "" {
			return 0, ErrMailTaskNotRetryable
		}
		create := db.EntClient.MailTask.Create().
			SetKind(e.Kind).
			SetRecipients(e.Recipients).
			SetSubject(e.Subject).
			SetBody(e.Body).
			SetNillableOrderID(e.OrderID).
			SetNillableItemID(e.ItemID)
		if e.Attachments != nil {
			create = create.SetAttachments(*e.Attachments)
		}
		copied, err := create.Save(ctx)
		if err != nil {
			return 0, err
		}
		id = copied.ID
	default:
		return 0, ErrMailTaskNotRetryable
	}
	go func(id int) { _ = db.DeliverMailTask(id) }(id)
	return id, nil
}

// RetryFailedMailTasks queues all failed tasks matching the filter again and
// returns how many were queued
func (db *Database) RetryFailedMailTasks(f MailTaskFilter) (int, error) {
	f.Status = MailTaskFailed
	n, err := db.EntClient.MailTask.Update().
		Where(mailTaskPredicates(f, true)...).
		SetStatus(MailTaskPending).
		SetAttempts(0).
		SetNextAttemptAt(time.Now()).
		Save(context.Background())
	if err != nil {
		return 0, err
	}
	if n > 0 {
		go func() { _, _ = db.ProcessDueMailTasks(n) }()
	}
	return n, nil
}

// MailRecipientsForItem returns the recipients that already have a mail task of the
// given kind for the item, whatever its state
func (db *Database) MailRecipientsForItem(itemID int, kind string) (map[string]bool, error) {
	rows, err := db.EntClient.MailTask.Query().
		Where(mailtask.ItemID(itemID), mailtask.KindEQ(kind)).
		Select(mailtask.FieldRecipients).
		Strings(context.Background())
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		for _, addr := range splitRecipients(r) {
			out[strings.ToLower(addr)] = true
		}
	}
	return out, nil
}

// MailDeliveryStatus sums up the mail tasks of one kind for an item
type MailDeliveryStatus struct {
	Counts     map[string]int
	Total      int
	FirstQueuedAt null.Time `swaggertype:"string"`
	LastSentAt   null.Time `swaggertype:"string"`
}

// GetMailDeliveryStatusForItem sums up the mail tasks of the given kind for the item
func (db *Database) GetMailDeliveryStatusForItem(itemID int, kind string) (MailDeliveryStatus, error) {
	status := MailDeliveryStatus{Counts: map[string]int{
		MailTaskPending: 0, MailTaskSending: 0, MailTaskSent: 0, MailTaskFailed: 0,
	}}
	rows, err := db.EntClient.MailTask.Query().
		Where(mailtask.ItemID(itemID), mailtask.KindEQ(kind)).
		Select(mailtask.FieldStatus, mailtask.FieldCreatedAt, mailtask.FieldSentAt).
		All(context.Background())
	if err != nil {
		return status, err
	}
	for _, r := range rows {
		status.Counts[r.Status]++
		status.Total++
		if !status.FirstQueuedAt.Valid || r.CreatedAt.Before(status.FirstQueuedAt.Time) {
			status.FirstQueuedAt = null.TimeFrom(r.CreatedAt)
		}
		if r.SentAt != nil && (!status.LastSentAt.Valid || r.SentAt.After(status.LastSentAt.Time)) {
			status.LastSentAt = null.TimeFrom(*r.SentAt)
		}
	}
	return status, nil
}
