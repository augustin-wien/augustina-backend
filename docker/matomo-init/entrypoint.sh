#!/bin/sh
# Runs the Matomo installer so nobody has to click through it.
#
# Matomo has no unattended CLI install: before it is installed the console does not even offer
# core:create-superuser or site:add, so the only way in is the web installer. It is a plain
# sequence of POST forms without CSRF tokens, which is what this script walks through.
#
# Everything is idempotent — on an already installed instance only the trusted hosts are checked,
# so it is safe to leave the service in the compose file and restart it any time.
set -e

BASE="${MATOMO_URL:-http://matomo}/index.php"
JAR=/tmp/matomo-cookies
CONFIG=/var/www/html/config/config.ini.php

step() {
  echo "  -> $1"
}

# While Matomo is not installed every page is the installer, and the installer says so in a flag
# it puts on the window object. The page title carries the version number, so it is no good here.
installed() {
  ! curl -s "$BASE" | grep -q "installation: true"
}

# The install runs against the internal hostname, so that is the only host Matomo ends up
# trusting. The browser reaches it under a different one and would get a "not a trusted host"
# warning instead of a login — and so would the tracker requests coming from the shop. Add the
# outside host rather than turning the check off.
trust_host() {
  [ -n "$1" ] || return 0
  grep -q "trusted_hosts\[\] = \"$1\"" "$CONFIG" && return 0

  step "trusting host $1"
  sed -i "/^\[General\]/a trusted_hosts[] = \"$1\"" "$CONFIG"
  # Matomo keeps a parsed copy of the config on disk.
  rm -rf /var/www/html/tmp/cache/tracker/* /var/www/html/tmp/templates_c/* 2>/dev/null || true
}

apply_trusted_hosts() {
  if [ ! -f "$CONFIG" ]; then
    echo "  !! ${CONFIG} not found — mount the matomo volume to set trusted hosts."
    return 0
  fi

  # http://localhost:8092 -> localhost:8092
  trust_host "$(echo "${MATOMO_URL_EXTERNAL:-}" | sed -e 's|^https\?://||' -e 's|/.*$||')"
}

echo "Waiting for Matomo..."
until curl -sf -o /dev/null "$BASE"; do
  sleep 2
done

if installed; then
  echo "Matomo is already installed."
  apply_trusted_hosts
  exit 0
fi

echo "Installing Matomo..."
rm -f $JAR

curl -s -c $JAR -b $JAR "$BASE?action=systemCheck" -o /dev/null

# The image writes the credentials into config.ini.php from MATOMO_DATABASE_*, but the installer
# keeps its own copy in the session, so they have to be posted once more. adapter and schema are
# required — without them the form silently re-renders and no tables are created.
step "database"
curl -s -c $JAR -b $JAR "$BASE?action=databaseSetup" \
  --data-urlencode "type=InnoDB" \
  --data-urlencode "host=${MATOMO_DATABASE_HOST}" \
  --data-urlencode "username=${MATOMO_DATABASE_USERNAME}" \
  --data-urlencode "password=${MATOMO_DATABASE_PASSWORD}" \
  --data-urlencode "dbname=${MATOMO_DATABASE_DBNAME}" \
  --data-urlencode "tables_prefix=matomo_" \
  --data-urlencode 'adapter=PDO\MYSQL' \
  --data-urlencode "schema=${MATOMO_DATABASE_SCHEMA:-Mariadb}" \
  --data-urlencode "submit=Next" -o /dev/null

step "tables"
curl -s -c $JAR -b $JAR "$BASE?action=tablesCreation" -o /dev/null

step "superuser ${MATOMO_ADMIN_USER}"
curl -s -c $JAR -b $JAR "$BASE?action=setupSuperUser" \
  --data-urlencode "login=${MATOMO_ADMIN_USER}" \
  --data-urlencode "password=${MATOMO_ADMIN_PASSWORD}" \
  --data-urlencode "password_bis=${MATOMO_ADMIN_PASSWORD}" \
  --data-urlencode "email=${MATOMO_ADMIN_EMAIL}" \
  --data-urlencode "subscribe_newsletter_piwikorg=0" \
  --data-urlencode "subscribe_newsletter_professionalservices=0" \
  --data-urlencode "submit=Next" -o /dev/null

step "site ${MATOMO_SITE_NAME} (${MATOMO_SITE_URL})"
curl -s -c $JAR -b $JAR "$BASE?action=firstWebsiteSetup" \
  --data-urlencode "siteName=${MATOMO_SITE_NAME}" \
  --data-urlencode "url=${MATOMO_SITE_URL}" \
  --data-urlencode "timezone=${MATOMO_SITE_TIMEZONE:-Europe/Vienna}" \
  --data-urlencode "ecommerce=0" \
  --data-urlencode "submit=Next" -o /dev/null

# anonymise_ip is switched on deliberately: it is part of the configuration that lets the shop
# track cookieless without a consent banner.
step "finish"
curl -s -c $JAR -b $JAR "$BASE?action=trackingCode" -o /dev/null
curl -s -c $JAR -b $JAR "$BASE?action=finished" \
  --data-urlencode "anonymise_ip=1" \
  --data-urlencode "setup_geoip2=0" \
  --data-urlencode "submit=Continue" -o /dev/null

if ! installed; then
  echo "Matomo installation did not finish — open ${MATOMO_URL_EXTERNAL:-$MATOMO_URL} and check."
  exit 1
fi

apply_trusted_hosts

echo "Matomo installed."
echo "  UI:    ${MATOMO_URL_EXTERNAL:-$MATOMO_URL}"
echo "  Login: ${MATOMO_ADMIN_USER} / ${MATOMO_ADMIN_PASSWORD}"
echo "  Site id 1 — enter it with the url under Backoffice > Settings > Matomo."
