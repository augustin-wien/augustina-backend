package mailer

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

// fakeSMTP is a minimal SMTP server that advertises authMech (none if empty)
// and rejects the command named by failAt.
type fakeSMTP struct {
	authMech string
	failAt   string
}

func (f fakeSMTP) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	reply := func(s string) {
		fmt.Fprint(w, s+"\r\n")
		_ = w.Flush()
	}
	fail := func(cmd, ok, bad string) {
		if f.failAt == cmd {
			reply(bad)
		} else {
			reply(ok)
		}
	}
	reply("220 localhost fake")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			if f.authMech != "" {
				reply("250-localhost")
				reply("250 AUTH " + f.authMech)
			} else {
				reply("250 localhost")
			}
		case strings.HasPrefix(cmd, "AUTH LOGIN"):
			if f.failAt == "AUTH" {
				reply("535 authentication failed")
				continue
			}
			reply("334 VXNlcm5hbWU6")
			_, _ = r.ReadString('\n')
			reply("334 UGFzc3dvcmQ6")
			_, _ = r.ReadString('\n')
			reply("235 ok")
		case strings.HasPrefix(cmd, "AUTH"):
			fail("AUTH", "235 ok", "535 authentication failed")
		case strings.HasPrefix(cmd, "MAIL"):
			fail("MAIL", "250 ok", "550 sender rejected")
		case strings.HasPrefix(cmd, "RCPT"):
			fail("RCPT", "250 ok", "550 recipient rejected")
		case cmd == "DATA":
			if f.failAt == "DATA" {
				reply("554 no data")
				continue
			}
			reply("354 go ahead")
			for {
				l, err := r.ReadString('\n')
				if err != nil || l == ".\r\n" {
					break
				}
			}
			fail("BODY", "250 queued", "554 message rejected")
		case cmd == "QUIT":
			fail("QUIT", "221 bye", "500 quit failed")
			return
		default:
			reply("250 ok")
		}
	}
}

// startFakeSMTP starts the server (with implicit TLS if useTLS) and points the
// mailer configuration at it.
func startFakeSMTP(t *testing.T, f fakeSMTP, useTLS bool, username string) {
	t.Helper()
	origConfig := config.Config
	origAuth := auth
	t.Cleanup(func() {
		config.Config = origConfig
		auth = origAuth
	})

	var l net.Listener
	var err error
	if useTLS {
		cert, cerr := generateSelfSignedCert("localhost")
		require.NoError(t, cerr)
		l, err = tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	} else {
		l, err = net.Listen("tcp", "127.0.0.1:0")
	}
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go f.serve(conn)
		}
	}()

	_, port, _ := net.SplitHostPort(l.Addr().String())
	config.Config.SMTPServer = "localhost"
	config.Config.SMTPPort = port
	config.Config.SMTPUsername = username
	config.Config.SMTPPassword = "secret"
	config.Config.SMTPSenderAddress = "sender@test.local"
	config.Config.SMTPSsl = useTLS
	config.Config.SMTPInsecureSkipVerify = true
	config.Config.Development = true
	config.Config.AllowInsecureSMTP = true
	Init()
}

func TestSendEmailServerAnswers(t *testing.T) {
	cases := []struct {
		name     string
		server   fakeSMTP
		username string
		wantErr  bool
	}{
		{"no auth configured", fakeSMTP{}, "", false},
		{"auth plain", fakeSMTP{authMech: "PLAIN"}, "user", false},
		{"auth login", fakeSMTP{authMech: "LOGIN"}, "user", false},
		{"unknown mechanism falls back to plain", fakeSMTP{authMech: "CRAM-MD5"}, "user", false},
		{"server without auth extension", fakeSMTP{}, "user", false},
		{"server without auth extension rejects auth", fakeSMTP{failAt: "AUTH"}, "user", true},
		{"xoauth2 only", fakeSMTP{authMech: "XOAUTH2"}, "user", true},
		{"auth rejected", fakeSMTP{authMech: "PLAIN", failAt: "AUTH"}, "user", true},
		{"login rejected", fakeSMTP{authMech: "LOGIN", failAt: "AUTH"}, "user", true},
		{"sender rejected", fakeSMTP{failAt: "MAIL"}, "", true},
		{"recipient rejected", fakeSMTP{failAt: "RCPT"}, "", true},
		{"data rejected", fakeSMTP{failAt: "DATA"}, "", true},
		{"message rejected", fakeSMTP{failAt: "BODY"}, "", true},
		{"quit fails", fakeSMTP{failAt: "QUIT"}, "", true},
	}
	for _, useTLS := range []bool{false, true} {
		for _, tc := range cases {
			name := tc.name
			if useTLS {
				name = "tls/" + name
			}
			t.Run(name, func(t *testing.T) {
				startFakeSMTP(t, tc.server, useTLS, tc.username)
				req, err := NewRequest([]string{"to@test.local"}, "Betreff", "<p>Hallo</p>")
				require.NoError(t, err)
				ok, err := req.SendEmail()
				if tc.wantErr {
					require.Error(t, err)
					require.False(t, ok)
					return
				}
				require.NoError(t, err)
				require.True(t, ok)
			})
		}
	}
}

func TestSendEmailUnreachableServer(t *testing.T) {
	origConfig := config.Config
	t.Cleanup(func() { config.Config = origConfig })
	config.Config.SMTPServer = "127.0.0.1"
	config.Config.SMTPPort = "1"
	config.Config.SMTPSenderAddress = "sender@test.local"

	req, err := NewRequest([]string{"to@test.local"}, "s", "b")
	require.NoError(t, err)

	config.Config.SMTPSsl = false
	_, err = req.SendEmail()
	require.Error(t, err)

	config.Config.SMTPSsl = true
	_, err = req.SendEmail()
	require.Error(t, err)
}
