#!/bin/bash
# Generates a fake Firebase service account JSON for integration tests.
# The RSA key is throwaway — it only needs to be valid so the Firebase SDK can sign JWTs.
# WireMock does not validate these tokens.

set -e

OUTFILE="${1:-firebase-credentials.json}"

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
