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
    "crypto/tls"

    "github.com/augustin-wien/augustina-backend/config"
)

// This integration test starts a fake SMTP server that advertises STARTTLS, then upgrades
// to TLS when the client issues STARTTLS and continues the SMTP session over TLS.
func TestSendEmail_Integration_STARTTLS(t *testing.T) {
    l, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("failed to listen: %v", err)
    }
    defer l.Close()
    addr := l.Addr().String()
    parts := strings.Split(addr, ":")
    port := parts[len(parts)-1]

    // configure mailer
    config.Config.SMTPServer = "localhost"
    config.Config.SMTPPort = port
    config.Config.SMTPUsername = "starttlsuser"
    config.Config.SMTPPassword = "starttlspass"
    config.Config.SMTPSenderAddress = "sender@test.local"
    config.Config.SMTPSsl = false
    config.Config.SMTPInsecureSkipVerify = true

    msgCh := make(chan string, 1)

    go func() {
        conn, err := l.Accept()
        if err != nil {
            t.Logf("accept error: %v", err)
            return
        }
        // do not defer close here; we'll close at the end of the TLS session

        r := bufio.NewReader(conn)
        w := bufio.NewWriter(conn)

        // greet
        fmt.Fprint(w, "220 localhost smtp fake server\r\n")
        w.Flush()

        // read EHLO/HELO
        _, _ = r.ReadString('\n')
        // advertise STARTTLS
        fmt.Fprint(w, "250-localhost Hello\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
        w.Flush()

        // wait for STARTTLS
        for {
            line, err := r.ReadString('\n')
            if err != nil {
                return
            }
            cmd := strings.TrimSpace(line)
            if strings.EqualFold(cmd, "STARTTLS") {
                fmt.Fprint(w, "220 Ready to start TLS\r\n")
                w.Flush()

                // upgrade server side to TLS
                cert, cerr := generateSelfSignedCert("localhost")
                if cerr != nil {
                    t.Logf("failed to generate cert: %v", cerr)
                    conn.Close()
                    return
                }
                tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
                tlsConn := tls.Server(conn, tlsConfig)
                if err := tlsConn.Handshake(); err != nil {
                    t.Logf("tls handshake failed: %v", err)
                    tlsConn.Close()
                    return
                }

                // continue over TLS
                r = bufio.NewReader(tlsConn)
                w = bufio.NewWriter(tlsConn)

                // expect EHLO again after STARTTLS
                _, _ = r.ReadString('\n')
                fmt.Fprint(w, "250-localhost Hello\r\n250 AUTH PLAIN\r\n")
                w.Flush()

                // handle AUTH, MAIL, RCPT, DATA
                for {
                    line, err := r.ReadString('\n')
                    if err != nil {
                        tlsConn.Close()
                        return
                    }
                    cmd := strings.TrimSpace(line)
                    if strings.HasPrefix(cmd, "AUTH PLAIN") {
                        parts := strings.SplitN(cmd, " ", 3)
                        var payload string
                        if len(parts) == 3 {
                            payload = parts[2]
                        } else {
                            next, _ := r.ReadString('\n')
                            payload = strings.TrimSpace(next)
                        }
                        _, derr := base64.StdEncoding.DecodeString(payload)
                        if derr == nil {
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
                        var sb strings.Builder
                        for {
                            l, _ := r.ReadString('\n')
                            if l == ".\r\n" {
                                break
                            }
                            sb.WriteString(l)
                        }
                        fmt.Fprint(w, "250 OK queued\r\n")
                        w.Flush()
                        msgCh <- sb.String()
                        continue
                    }
                    if strings.HasPrefix(cmd, "QUIT") {
                        fmt.Fprint(w, "221 Bye\r\n")
                        w.Flush()
                        tlsConn.Close()
                        return
                    }
                    fmt.Fprint(w, "250 OK\r\n")
                    w.Flush()
                }
            }
            // ignore other commands until STARTTLS
        }
    }()

    // init and send
    Init()
    rreq := NewRequest([]string{"recipient@test.local"}, "STARTTLS Integration Test", "Hello STARTTLS Integration")
    ok, err := rreq.SendEmail()
    if err != nil {
        t.Fatalf("SendEmail returned error: %v", err)
    }
    if !ok {
        t.Fatalf("SendEmail returned false")
    }

    select {
    case msg := <-msgCh:
        if !strings.Contains(msg, "STARTTLS Integration Test") || !strings.Contains(msg, "Hello STARTTLS Integration") {
            t.Fatalf("message content mismatch: %q", msg)
        }
    case <-time.After(5 * time.Second):
        t.Fatalf("timeout waiting for message on fake starttls smtp server")
    }
}
