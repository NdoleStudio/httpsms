#!/usr/bin/env bash
set -euo pipefail

export MSYS2_ARG_CONV_EXCL="/CN="

output_dir="${1:-certs}"
mkdir -p "$output_dir"

openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "$output_dir/ca-key.pem" \
  -out "$output_dir/ca.pem" \
  -days 2 \
  -subj "/CN=httpSMS integration adapter CA"

openssl req -newkey rsa:2048 -nodes \
  -keyout "$output_dir/server-key.pem" \
  -out "$output_dir/server.csr" \
  -subj "/CN=adapter-emulator"

cat >"$output_dir/server.ext" <<'EOF'
subjectAltName=DNS:adapter-emulator
extendedKeyUsage=serverAuth
EOF

openssl x509 -req \
  -in "$output_dir/server.csr" \
  -CA "$output_dir/ca.pem" \
  -CAkey "$output_dir/ca-key.pem" \
  -CAcreateserial \
  -out "$output_dir/server.pem" \
  -days 2 \
  -extfile "$output_dir/server.ext"

# The emulator runs as an unprivileged container user and must read this
# bind-mounted throwaway key.
chmod 0644 "$output_dir/server-key.pem"
