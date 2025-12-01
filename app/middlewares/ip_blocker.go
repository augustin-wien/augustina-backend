package middlewares

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/augustin-wien/augustina-backend/utils"
)

// IPBlocker manages blocked IPs
type IPBlocker struct {
	mu         sync.RWMutex
	blockedIPs map[string]time.Time
}

var (
	// GlobalBlocker instance
	GlobalBlocker = &IPBlocker{
		blockedIPs: make(map[string]time.Time),
	}

	// Paths that trigger an immediate block
	suspiciousPaths = []string{
		"/robots.txt",
		"/.env",
		"/.ds_store",
		"/login.action",
		"/v2/_catalog",
		"/wp-login.php",
		"/admin.php",
		"/config.json",
		"/git/config",
		"/aws/credentials",
		"/info.php",
		"/telescope/requests",
		"/meta-inf/maven/",
		"/_all_dbs",
		"/server-status",
		"/ecp/",
		"/debug/default/view",
		"/about",
		"/.vscode/sftp.json",
		"/version",
		"/server",
		"/actuator/env",
		"/@vite/env",
		"/api/swagger.json",
		"/api-docs/swagger.json",
		"/v3/api-docs",
		"/v2/api-docs",
		"/swagger/v1/swagger.json",
		"/swagger.json",
		"/webjars/swagger-ui/index.html",
		"/swagger/swagger-ui.html",
		"/swagger/index.html",
		"/swagger-ui.html",
		"/api/gql",
		"/graphql/api",
		"/api/graphql",
		"/graphql",
		"/phpinfo.php",
		"database.env",

		// Version Control & IDEs
		"/.git/",
		"/.svn/",
		"/.hg/",
		"/.idea/",
		"/.vscode/",
		"/sftp-config.json",

		// Cloud & Container Metadata
		"/aws/config",
		"/.aws/",
		"/config/config.json",
		"/docker-compose.yml",
		"/docker-compose.yaml",
		"/kube-system",
		"/.kube/config",

		// Common Backups & Archives
		".bak",
		".old",
		".swp",
		".sql",
		".zip",
		".tar.gz",
		".tgz",
		"backup.sql",
		"dump.sql",

		// CMS & Admin Panels
		"/wp-admin",
		"/wp-content",
		"/wp-includes",
		"/xmlrpc.php",
		"/phpmyadmin",
		"/pma/",
		"/admin/",
		"/administrator/",
		"/console/",
		"/dashboard/",

		// Framework Specifics
		"/actuator/",
		"/jolokia",
		"/env",
		"/_profiler",
		"/telescope",
		"/debug/vars",

		// Shells & Remote Execution
		"/shell.php",
		"/cmd.php",
		"/c99.php",
		"/r57.php",
		"/cgi-bin/",
		"/bin/sh",
		"/bin/bash",
	}

	// Paths that trigger a block only on exact match
	suspiciousExactPaths = []string{
		"/api",
	}

	// Blocked User-Agents (Bots, Scrapers, AI Agents)
	blockedUserAgents = []string{
		"curl",
		"wget",
		"python",
		"scrapy",
		"http-client",
		"postman",
		"aiohttp",
		"httpx",
		"go-http-client",
		"bot",
		"crawler",
		"spider",
	}
)

// BlockBadUserAgents middleware blocks requests from non-browser tools
func BlockBadUserAgents(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow health checks to pass through without UA checks (often done by curl/kube-probe)
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		ua := strings.ToLower(r.UserAgent())
		for _, badUA := range blockedUserAgents {
			if strings.Contains(ua, badUA) {
				log.Warnf("Security: Blocked User-Agent %s from %s", r.UserAgent(), utils.ReadUserIP(r))
				// Tarpit: Delay response to slow down the bot
				time.Sleep(2 * time.Second)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// BlockIP blocks an IP for a specific duration
func (b *IPBlocker) BlockIP(ip string, duration time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blockedIPs[ip] = time.Now().Add(duration)
	log.Warnf("Security: Blocked IP %s for %v due to suspicious activity", ip, duration)
}

// IsBlocked checks if an IP is currently blocked
func (b *IPBlocker) IsBlocked(ip string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	expiry, exists := b.blockedIPs[ip]
	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		return false
	}
	return true
}

// FilterBlockedIPs middleware rejects requests from blocked IPs
func FilterBlockedIPs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := utils.ReadUserIP(r)
		if GlobalBlocker.IsBlocked(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// BlockSuspiciousRequests middleware checks for malicious paths and blocks the IP
func BlockSuspiciousRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.ToLower(r.URL.Path)

		for _, suspicious := range suspiciousPaths {
			if strings.Contains(path, suspicious) {
				ip := utils.ReadUserIP(r)
				GlobalBlocker.BlockIP(ip, 24*time.Hour)

				// Tarpit: Slow down the attacker significantly
				time.Sleep(5 * time.Second)

				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		for _, suspicious := range suspiciousExactPaths {
			if path == suspicious {
				ip := utils.ReadUserIP(r)
				GlobalBlocker.BlockIP(ip, 24*time.Hour)

				// Tarpit: Slow down the attacker significantly
				time.Sleep(5 * time.Second)

				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
