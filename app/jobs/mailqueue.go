package jobs

import (
	"time"

	"github.com/augustin-wien/augustina-backend/database"
)

// mailQueueInterval is how often due mail tasks are sent
const mailQueueInterval = 30 * time.Second

// mailQueueBatch is how many mail tasks one run sends at most
const mailQueueBatch = 50

// mailContentRetention is how long body and attachments of a sent mail are kept.
// They contain personal download and login links; the delivery record stays.
const mailContentRetention = 30 * 24 * time.Hour

// mailPurgeInterval is how often the content of old sent mails is removed
const mailPurgeInterval = 24 * time.Hour

// StartMailQueue runs a background job that sends queued mails whose next attempt is
// due, i.e. retries of failed attempts and mails left over by a restart, and removes
// the content of old sent mails.
func StartMailQueue() {
	go func() {
		processMailQueue()
		purgeSentMailContent()
		lastPurge := time.Now()

		ticker := time.NewTicker(mailQueueInterval)
		for now := range ticker.C {
			processMailQueue()
			if now.Sub(lastPurge) >= mailPurgeInterval {
				purgeSentMailContent()
				lastPurge = now
			}
		}
	}()
}

func processMailQueue() {
	for {
		n, err := database.Db.ProcessDueMailTasks(mailQueueBatch)
		if err != nil || n < mailQueueBatch {
			return
		}
	}
}

func purgeSentMailContent() {
	n, err := database.Db.PurgeSentMailContent(time.Now().Add(-mailContentRetention))
	if err != nil {
		log.Errorw("purgeSentMailContent: failed", "error", err)
		return
	}
	if n > 0 {
		log.Infow("purgeSentMailContent: removed content of old sent mails", "count", n)
	}
}
