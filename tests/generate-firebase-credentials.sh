#!/bin/bash
# Generates the throwaway credentials the integration stack needs:
#
#   1. a fake Firebase service account JSON (consumed by the API through the
#      FIREBASE_CREDENTIALS environment variable), and
#   2. the test-only RSA signing key, self-signed certificate, and WireMock
#      Firebase certificate mapping the MCP service and the MCP integration
#      tests share.
#
# Every artifact is disposable and is regenerated on each invocation. None of
# them is ever committed: see tests/.gitignore.
#
# The RSA key in the service account JSON is throwaway — it only needs to be
# valid so the Firebase SDK can sign JWTs. WireMock does not validate those
# tokens.
#
# The MCP signing key is used for two things, both test-only:
#
#   * the MCP service signs its own MCP access tokens and downstream API
#     delegation tokens with it (MCP_SIGNING_PRIVATE_KEY_FILE), publishing the
#     matching public key at /.well-known/jwks.json for the API to verify
#     against, and
#   * the integration tests sign deterministic Firebase-style ID tokens with
#     it, which the MCP service verifies against the self-signed certificate
#     served by WireMock at /firebase-certs under the "mcp-test-key" key ID.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

OUTFILE="${1:-firebase-credentials.json}"

MCP_SIGNING_KEY="$SCRIPT_DIR/mcp-test-signing-key.pem"
MCP_SIGNING_CERT="$SCRIPT_DIR/mcp-test-signing-cert.pem"
MCP_CERTS_MAPPING="$SCRIPT_DIR/wiremock/mappings/firebase-certs.generated.json"

# The key ID published in the WireMock Firebase certificate map, used as the
# "kid" header of every test Firebase ID token and of every token the MCP
# service mints (MCP_SIGNING_KEY_ID in tests/docker-compose.yml).
MCP_SIGNING_KEY_ID="mcp-test-key"

# Generate a 2048-bit RSA key
PRIVATE_KEY=$(openssl genrsa 2048 2>/dev/null)

# Escape newlines for JSON embedding
PRIVATE_KEY_ESCAPED=$(echo "$PRIVATE_KEY" | awk '{printf "%s\\n", $0}')

cat > "$OUTFILE" <<EOF
{
  "type": "service_account",
  "project_id": "httpsms-test",
  "private_key_id": "test-key-id",
  "private_key": "${PRIVATE_KEY_ESCAPED}",
  "client_email": "test@httpsms-test.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "http://wiremock:8080/auth",
  "token_uri": "http://wiremock:8080/token",
  "auth_provider_x509_cert_url": "http://wiremock:8080/certs",
  "client_x509_cert_url": "http://wiremock:8080/certs/test"
}
EOF

echo "Generated $OUTFILE"

# Generate the MCP signing key and its self-signed certificate, so the
# certificate WireMock serves always matches the key the MCP container mounts
# and the tests sign with. The key is written unencrypted (PKCS#1), which is
# what MCP_SIGNING_PRIVATE_KEY_FILE expects.
#
# The subject is supplied through a generated config file rather than -subj:
# Git Bash on Windows rewrites a "/CN=..." argument into a Windows path, and
# disabling that rewriting would equally break the key/certificate output
# paths. A config file is handled correctly on every platform.
OPENSSL_CONF_FILE="$SCRIPT_DIR/mcp-test-openssl.cnf"
cat > "$OPENSSL_CONF_FILE" <<'EOF'
[req]
distinguished_name = dn
prompt = no
x509_extensions = ext

[dn]
CN = httpsms-mcp-integration-tests

[ext]
basicConstraints = critical,CA:FALSE
EOF

openssl genrsa -out "$MCP_SIGNING_KEY" 2048 2>/dev/null
openssl req -x509 -new -key "$MCP_SIGNING_KEY" -days 3650 \
  -config "$OPENSSL_CONF_FILE" \
  -out "$MCP_SIGNING_CERT" 2>/dev/null

rm -f "$OPENSSL_CONF_FILE"

# The MCP container runs as the unprivileged "mcp" user and mounts the key
# read-only, so it must be world readable. This is a throwaway test key that
# never leaves the developer machine or the CI runner.
chmod 0644 "$MCP_SIGNING_KEY" "$MCP_SIGNING_CERT"

echo "Generated $MCP_SIGNING_KEY"
echo "Generated $MCP_SIGNING_CERT"

# Emit the WireMock stub for the Firebase certificate endpoint the MCP service
# verifies test Firebase ID tokens against. The body is the flat
# "key ID -> PEM certificate" map Google's securetoken endpoint serves (not a
# JWKS document), which is exactly what the MCP service's Firebase certificate
# cache parses.
CERT_ESCAPED=$(awk '{printf "%s\\n", $0}' "$MCP_SIGNING_CERT")

cat > "$MCP_CERTS_MAPPING" <<EOF
{
  "request": {
    "urlPath": "/firebase-certs",
    "method": "GET"
  },
  "response": {
    "status": 200,
    "headers": {
      "Content-Type": "application/json"
    },
    "jsonBody": {
      "${MCP_SIGNING_KEY_ID}": "${CERT_ESCAPED}"
    }
  }
}
EOF

echo "Generated $MCP_CERTS_MAPPING"
