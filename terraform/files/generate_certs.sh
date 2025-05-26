#!/bin/bash

set -e

CERT_DIR=$1
CN_PREFIX=${2:-opensearch}
NAMESPACE=${3:-default}

mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

#  OpenSearch CA
openssl genrsa -out root-ca-key.pem 2048
openssl req -new -x509 -sha256 -key root-ca-key.pem -subj "/CN=${CN_PREFIX}-ca" -out root-ca.pem -days 730

#  Node Cert
cat > "${CN_PREFIX}.cnf" <<EOF
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
CN = ${CN_PREFIX}-cluster-master

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${CN_PREFIX}
DNS.2 = ${CN_PREFIX}-cluster-master
DNS.3 = ${CN_PREFIX}-cluster-master.${NAMESPACE}
DNS.4 = ${CN_PREFIX}-cluster-master.${NAMESPACE}.svc
DNS.5 = ${CN_PREFIX}-cluster-master.${NAMESPACE}.svc.cluster.local
DNS.6 = localhost
IP.1 = 127.0.0.1
EOF

openssl genrsa -out node-key.pem 2048
openssl req -new -key node-key.pem -out node.csr -config "${CN_PREFIX}.cnf"
openssl x509 -req -in node.csr -CA root-ca.pem -CAkey root-ca-key.pem -CAcreateserial -sha256 -out node.pem -days 730 -extensions req_ext -extfile "${CN_PREFIX}.cnf"

rm node.csr "${CN_PREFIX}.cnf" root-ca.srl

echo "Certificates generated in $CERT_DIR for namespace $NAMESPACE" 
echo ":D"