#!/bin/bash
# Sets up the Keycloak side of the Nextcloud login for one realm. Idempotent, so it can run on
# every start and also once by hand against a production Keycloak:
#
#   docker run --rm -v ./docker/nextcloud:/scripts --entrypoint /scripts/keycloak-client.sh \
#     -e KEYCLOAK_URL=https://login.example.org -e KEYCLOAK_ADMIN=... -e KEYCLOAK_ADMIN_PASSWORD=... \
#     -e KEYCLOAK_REALM=augustin -e NEXTCLOUD_PUBLIC_URL=https://cloud.example.org \
#     -e NEXTCLOUD_OIDC_CLIENT_SECRET=... quay.io/keycloak/keycloak:26.8.1
#
# It creates a confidential client, a client role that the realm roles admin and backoffice
# include, and a mapper that puts that role into the nextcloud_groups claim. Nextcloud turns the
# claim into a group and only lets members of that group log in. The realm role names are not
# passed on directly because Keycloak's "admin" would become Nextcloud's admin group.
set -euo pipefail

KCADM=/opt/keycloak/bin/kcadm.sh
REALM="${KEYCLOAK_REALM:?}"
CLIENT_ID="${NEXTCLOUD_OIDC_CLIENT_ID:-nextcloud}"
SECRET="${NEXTCLOUD_OIDC_CLIENT_SECRET:?}"
ROLE="${NEXTCLOUD_OIDC_ROLE:-augustin-backoffice}"
REALM_ROLES="${NEXTCLOUD_OIDC_REALM_ROLES:-admin backoffice}"
NC_URL="${NEXTCLOUD_PUBLIC_URL:?}"
NC_URL="${NC_URL%/}"

echo "Waiting for Keycloak..."
until $KCADM config credentials --server "${KEYCLOAK_URL:?}" --realm master \
  --user "${KEYCLOAK_ADMIN:?}" --password "${KEYCLOAK_ADMIN_PASSWORD:?}" >/dev/null 2>&1; do
  sleep 3
done

client_settings=(
  -s enabled=true
  -s protocol=openid-connect
  -s publicClient=false
  -s standardFlowEnabled=true
  -s implicitFlowEnabled=false
  -s directAccessGrantsEnabled=false
  -s "secret=${SECRET}"
  -s "rootUrl=${NC_URL}"
  -s "redirectUris=[\"${NC_URL}/*\"]"
  -s "webOrigins=[\"${NC_URL}\"]"
  -s "attributes.\"post.logout.redirect.uris\"=${NC_URL}/*"
)

id=$($KCADM get clients -r "$REALM" -q "clientId=${CLIENT_ID}" --fields id --format csv --noquotes)
if [ -z "$id" ]; then
  echo "  -> creating client ${CLIENT_ID}"
  $KCADM create clients -r "$REALM" -s "clientId=${CLIENT_ID}" "${client_settings[@]}"
  id=$($KCADM get clients -r "$REALM" -q "clientId=${CLIENT_ID}" --fields id --format csv --noquotes)
else
  echo "  -> updating client ${CLIENT_ID}"
  $KCADM update "clients/${id}" -r "$REALM" "${client_settings[@]}"
fi

if ! $KCADM get "clients/${id}/roles/${ROLE}" -r "$REALM" >/dev/null 2>&1; then
  echo "  -> creating client role ${ROLE}"
  $KCADM create "clients/${id}/roles" -r "$REALM" -s "name=${ROLE}" \
    -s "description=May open the vendor documents in Nextcloud"
fi

for realm_role in $REALM_ROLES; do
  echo "  -> realm role ${realm_role} includes ${ROLE}"
  $KCADM add-roles -r "$REALM" --rname "$realm_role" --cclientid "$CLIENT_ID" --rolename "$ROLE"
done

if ! $KCADM get "clients/${id}/protocol-mappers/models" -r "$REALM" --fields name --format csv --noquotes |
  grep -qx nextcloud_groups; then
  echo "  -> creating mapper nextcloud_groups"
  $KCADM create "clients/${id}/protocol-mappers/models" -r "$REALM" \
    -s name=nextcloud_groups \
    -s protocol=openid-connect \
    -s protocolMapper=oidc-usermodel-client-role-mapper \
    -s "config.\"usermodel.clientRoleMapping.clientId\"=${CLIENT_ID}" \
    -s 'config."claim.name"=nextcloud_groups' \
    -s 'config."jsonType.label"=String' \
    -s 'config.multivalued=true' \
    -s 'config."id.token.claim"=true' \
    -s 'config."access.token.claim"=true' \
    -s 'config."userinfo.token.claim"=true'
fi

echo "Keycloak client ${CLIENT_ID} is ready."
