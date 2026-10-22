#!/bin/sh
# Runs as www-data before every start of the Nextcloud container (and can be copied into the
# hooks folder of a production instance). Idempotent.
#
# Login: user_oidc against the Keycloak realm. Only members of the group from the
# nextcloud_groups claim (see keycloak-client.sh) may log in, and the login page goes
# straight to Keycloak; local accounts still work under /login?direct=1.
#
# Documents: a team folder (groupfolders app) for that group, and a local service account in
# the same group that the backend uses to create one folder per vendor in it.
set -e

occ() {
  php /var/www/html/occ "$@"
}

GROUP="${NEXTCLOUD_OIDC_ROLE:-augustin-backoffice}"
ROOT="${NEXTCLOUD_VENDOR_ROOT:-Verkaufende}"

for app in user_oidc groupfolders; do
  occ app:list --output=json | grep -q "\"${app}\"" || occ app:install "$app"
  occ app:enable "$app" >/dev/null
done
occ app:disable firstrunwizard >/dev/null 2>&1 || true

# In development Keycloak runs on a private address and nothing uses HTTPS.
if [ "${NEXTCLOUD_DEVELOPMENT:-false}" = "true" ]; then
  occ config:system:set allow_local_remote_servers --value=true --type=boolean >/dev/null
  occ config:app:set user_oidc allow_insecure_http --value=true --type=boolean --lazy >/dev/null
fi

occ user_oidc:provider "${NEXTCLOUD_OIDC_PROVIDER:-Keycloak}" \
  --clientid="${NEXTCLOUD_OIDC_CLIENT_ID:-nextcloud}" \
  --clientsecret="${NEXTCLOUD_OIDC_CLIENT_SECRET:?}" \
  --discoveryuri="${KEYCLOAK_URL%/}/realms/${KEYCLOAK_REALM:?}/.well-known/openid-configuration" \
  --scope="openid email profile" \
  --unique-uid=0 \
  --mapping-uid=preferred_username \
  --mapping-display-name=name \
  --mapping-email=email \
  --mapping-groups=nextcloud_groups \
  --group-provisioning=1 \
  --group-whitelist-regex="/^${GROUP}\$/" \
  --group-restrict-login-to-whitelist=1 >/dev/null
occ config:app:set user_oidc allow_multiple_user_backends --value=0 --lazy >/dev/null

occ group:info "$GROUP" >/dev/null 2>&1 || occ group:add "$GROUP"

SERVICE_USER="${NEXTCLOUD_SERVICE_USER:?}"
export OC_PASS="${NEXTCLOUD_SERVICE_PASSWORD:?}"
if occ user:info "$SERVICE_USER" >/dev/null 2>&1; then
  occ user:resetpassword --password-from-env "$SERVICE_USER" >/dev/null
else
  occ user:add --password-from-env --display-name="Augustin Backend" "$SERVICE_USER"
fi
occ group:adduser "$GROUP" "$SERVICE_USER" 2>/dev/null || true

folder_id=$(occ groupfolders:list --output=json | php -r '
  foreach (json_decode(stream_get_contents(STDIN), true) as $f) {
    if ($f["mount_point"] === $argv[1]) { echo $f["id"]; break; }
  }' "$ROOT")
if [ -z "$folder_id" ]; then
  folder_id=$(occ groupfolders:create "$ROOT")
fi
occ groupfolders:group "$folder_id" "$GROUP" read write share delete >/dev/null

echo "Nextcloud: Keycloak login and team folder \"${ROOT}\" are set up."
