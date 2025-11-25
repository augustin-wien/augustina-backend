//go:build integration

package mailer

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
)

// This integration test spins up a minimal fake SMTP server (plain, no STARTTLS)
// and verifies that SendEmail successfully performs EHLO/AUTH/MAIL/RCPT/DATA.
func TestSendEmail_Integration_FakeSMTP(t *testing.T) {
	// pick a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	// configure mailer to point to fake server
	config.Config.SMTPServer = "127.0.0.1"
	config.Config.SMTPPort = port
	config.Config.SMTPUsername = "testuser"
	config.Config.SMTPPassword = "testpass"
	config.Config.SMTPSenderAddress = "sender@test.local"
	config.Config.SMTPSsl = false

	// channel to receive the email body
	msgCh := make(chan string, 1)

	// accept one connection and perform simple SMTP dialog
	go func() {
		conn, err := l.Accept()
		if err != nil {
			t.Logf("listener accept error: %v", err)
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)

		// greet
		fmt.Fprint(w, "220 localhost smtp fake server\r\n")
		w.Flush()

		// read EHLO/HELO
		line, _ := r.ReadString('\n')
		_ = line
		// respond without STARTTLS capability
		fmt.Fprint(w, "250-localhost Hello\r\n250 AUTH PLAIN\r\n")
		w.Flush()

		// handle commands
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.TrimSpace(line)
			if strings.HasPrefix(cmd, "AUTH PLAIN") {
				// AUTH PLAIN <base64>
				parts := strings.SplitN(cmd, " ", 3)
				var payload string
				if len(parts) == 3 {
					payload = parts[2]
				} else {
					// client may send just AUTH PLAIN then payload next
					next, _ := r.ReadString('\n')
					payload = strings.TrimSpace(next)
				}
				// accept any credentials for the test
				_, err := base64.StdEncoding.DecodeString(payload)
				if err == nil {
					fmt.Fprint(w, "235 Authentication succeeded\r\n")
				} else {
					fmt.Fprint(w, "535 Authentication failed\r\n")
				}
				w.Flush()
				continue
			}
			if strings.HasPrefix(cmd, "MAIL FROM:") {
				fmt.Fprint(w, "250 OK\r\n")
				w.Flush()
				continue
			}
			if strings.HasPrefix(cmd, "RCPT TO:") {
				fmt.Fprint(w, "250 OK\r\n")
				w.Flush()
				continue
			}
			if strings.HasPrefix(cmd, "DATA") {
				fmt.Fprint(w, "354 End data with <CR><LF>.<CR><LF>\r\n")
				w.Flush()
				// read data until single dot line
				var sb strings.Builder
				for {
					l, _ := r.ReadString('\n')
					if l == ".\r\n" {
						break
					}
					sb.WriteString(l)
				}
				// ack
				fmt.Fprint(w, "250 OK queued\r\n")
				w.Flush()
				msgCh <- sb.String()
				continue
			}
			if strings.HasPrefix(cmd, "QUIT") {
				fmt.Fprint(w, "221 Bye\r\n")
				w.Flush()
				return
			}
			// default
			fmt.Fprint(w, "250 OK\r\n")
			w.Flush()
		}
	}()

	// init mailer auth
	Init()

	// create and send email
	r := NewRequest([]string{"recipient@test.local"}, "Integration Test", "Hello Integration")
	ok, err := r.SendEmail()
	if err != nil {
		t.Fatalf("SendEmail returned error: %v", err)
	}
	if !ok {
		t.Fatalf("SendEmail returned false")
	}

	select {
	case msg := <-msgCh:
		if !strings.Contains(msg, "Integration Test") || !strings.Contains(msg, "Hello Integration") {
			t.Fatalf("message content mismatch: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for message on fake smtp server")
	}
}
