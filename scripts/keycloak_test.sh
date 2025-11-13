#!/usr/bin/env bash
set -euo pipefail

KC_URL="http://keycloak:8080"
KC_REALM="augustin"
KC_USER="test_user"
KC_ADMIN_USER="admin"
KC_ADMIN_PASS="admin"

# 1. Get admin token
TOKEN=$(curl -s \
  -d "client_id=admin-cli" \
  -d "username=${KC_ADMIN_USER}" \
  -d "password=${KC_ADMIN_PASS}" \
  -d "grant_type=password" \
  "${KC_URL}/realms/master/protocol/openid-connect/token" \
| jq -r .access_token)

# 2. Get user info (to extract ID)
USER_JSON=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "${KC_URL}/admin/realms/${KC_REALM}/users?username=${KC_USER}")

USER_ID=$(echo "$USER_JSON" | jq -r '.[0].id')

if [[ "$USER_ID" == "null" || -z "$USER_ID" ]]; then
  echo "User '${KC_USER}' not found in realm '${KC_REALM}'"
  exit 1
fi

echo "=== User Info ==="
echo "$USER_JSON" | jq '.[0]'

# 3. Get groups
echo "=== Groups ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "${KC_URL}/admin/realms/${KC_REALM}/users/${USER_ID}/groups" | jq

# 4. Get realm roles
echo "=== Realm Roles (direct) ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "${KC_URL}/admin/realms/${KC_REALM}/users/${USER_ID}/role-mappings/realm" | jq

# 5. Get effective role mappings (includes groups & composites)
echo "=== Effective Roles ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "${KC_URL}/admin/realms/${KC_REALM}/users/${USER_ID}/role-mappings" | jq
