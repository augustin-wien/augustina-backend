package mailer

import (
	"errors"
	"net/textproto"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// stubSend replaces Send with a stub answering with the given errors in turn
// and returns a pointer to the number of calls made.
func stubSend(t *testing.T, errs ...error) *int {
	origSend, origDelays := Send, retryDelays
	t.Cleanup(func() { Send, retryDelays = origSend, origDelays })
	retryDelays = []time.Duration{0, 0, 0}
	calls := 0
	Send = func(r *EmailRequest) (bool, error) {
		err := errs[calls]
		calls++
		return err == nil, err
	}
	return &calls
}

func TestSendWithRetry_RetriesTransientError(t *testing.T) {
	transient := &textproto.Error{Code: 432, Msg: "4.2.0 STOREDRV.Submission.Exception"}
	calls := stubSend(t, transient, transient, nil)

	r, _ := NewRequest([]string{"test@example.com"}, "subj", "body")
	success, err := SendWithRetry(r)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, 3, *calls)
}

func TestSendWithRetry_GivesUpAfterLastDelay(t *testing.T) {
	transient := &textproto.Error{Code: 451, Msg: "try again later"}
	calls := stubSend(t, transient, transient, transient, transient)

	r, _ := NewRequest([]string{"test@example.com"}, "subj", "body")
	success, err := SendWithRetry(r)
	require.ErrorIs(t, err, transient)
	require.False(t, success)
	require.Equal(t, 4, *calls)
}

func TestSendWithRetry_DoesNotRetryPermanentError(t *testing.T) {
	for _, permanent := range []error{
		&textproto.Error{Code: 550, Msg: "mailbox unavailable"},
		errors.New("dial tcp: connection refused"),
	} {
		calls := stubSend(t, permanent)

		r, _ := NewRequest([]string{"test@example.com"}, "subj", "body")
		success, err := SendWithRetry(r)
		require.ErrorIs(t, err, permanent)
		require.False(t, success)
		require.Equal(t, 1, *calls)
	}
}
