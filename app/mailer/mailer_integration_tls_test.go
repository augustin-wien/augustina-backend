//go:build integration

package mailer

import (
    "bufio"
    "crypto/rand"
    "crypto/rsa"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/base64"
    "encoding/pem"
    "fmt"
    "math/big"
    "net"
    "strings"
    "testing"
    "time"

    "github.com/augustin-wien/augustina-backend/config"
)

// This integration test spins up a minimal fake SMTP server over implicit TLS (SMTPS, port 465 style)
// and verifies that SendEmail successfully connects over TLS and performs EHLO/AUTH/MAIL/RCPT/DATA.
func TestSendEmail_Integration_FakeSMTPS_TLS(t *testing.T) {
    // pick a free port on loopback
    l, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("failed to listen: %v", err)
    }
    defer l.Close()
    addr := l.Addr().String()
    parts := strings.Split(addr, ":")
    port := parts[len(parts)-1]

    // generate a self-signed cert for localhost
    cert, err := generateSelfSignedCert("localhost")
    if err != nil {
        t.Fatalf("failed to create cert: %v", err)
    }

    tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
    tlsListener := tls.NewListener(l, tlsConfig)

    // configure mailer to point to fake server using hostname 'localhost' (matches cert)
    config.Config.SMTPServer = "localhost"
    config.Config.SMTPPort = port
    config.Config.SMTPUsername = "tlsuser"
    config.Config.SMTPPassword = "tlspass"
    config.Config.SMTPSenderAddress = "sender@test.local"
    config.Config.SMTPSsl = true
    // allow insecure verify for the self-signed test certificate
    config.Config.SMTPInsecureSkipVerify = true

    msgCh := make(chan string, 1)

    go func() {
        conn, err := tlsListener.Accept()
        if err != nil {
            t.Logf("accept error: %v", err)
            return
        }
        defer conn.Close()
        r := bufio.NewReader(conn)
        w := bufio.NewWriter(conn)

        // greet
        fmt.Fprint(w, "220 localhost smtp fake smtps server\r\n")
        w.Flush()

        // read EHLO/HELO
        _, _ = r.ReadString('\n')
        // respond advertising AUTH PLAIN
        fmt.Fprint(w, "250-localhost Hello\r\n250 AUTH PLAIN\r\n")
        w.Flush()

        for {
            line, err := r.ReadString('\n')
            if err != nil {
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
                // decode and accept any creds
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
                return
            }
            fmt.Fprint(w, "250 OK\r\n")
            w.Flush()
        }
    }()

    // init mailer and send
    Init()
    rreq := NewRequest([]string{"recipient@test.local"}, "TLS Integration Test", "Hello TLS Integration")
    ok, err := rreq.SendEmail()
    if err != nil {
        t.Fatalf("SendEmail error: %v", err)
    }
    if !ok {
        t.Fatalf("SendEmail returned false")
    }

    select {
    case msg := <-msgCh:
        if !strings.Contains(msg, "TLS Integration Test") || !strings.Contains(msg, "Hello TLS Integration") {
            t.Fatalf("message content mismatch: %q", msg)
        }
    case <-time.After(5 * time.Second):
        t.Fatalf("timeout waiting for message on fake smtps server")
    }
}

// generateSelfSignedCert returns a tls.Certificate for the given host (DNS name).
func generateSelfSignedCert(host string) (tls.Certificate, error) {
    priv, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return tls.Certificate{}, err
    }
    serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
    serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
    if err != nil {
        return tls.Certificate{}, err
    }
    tmpl := x509.Certificate{
        SerialNumber: serialNumber,
        Subject:      pkixName(host),
        NotBefore:    time.Now().Add(-time.Hour),
        NotAfter:     time.Now().Add(24 * time.Hour),
        KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        DNSNames:     []string{host},
        IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
    }
    derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
    if err != nil {
        return tls.Certificate{}, err
    }
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
    keyBytes := x509.MarshalPKCS1PrivateKey(priv)
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
    return tls.X509KeyPair(certPEM, keyPEM)
}

// pkixName is a tiny helper to create a reasonable pkix.Name without importing too many packages inline.
func pkixName(common string) (name pkix.Name) {
    name.CommonName = common
    return
}
