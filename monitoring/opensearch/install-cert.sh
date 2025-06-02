#!/bin/bash

gcloud compute ssh --zone "europe-west1-d" "it-school-2025-1" --tunnel-through-iap --project "fc-it-school-2025" << 'EOF'

CERT_DIR="./opensearch-certs"
NAMESPACE="dav-monitoring"
SECRET_NAME="opensearch-certs"

mkdir -p $CERT_DIR
cd $CERT_DIR

cat > opensearch.cnf << 'CONFEND'
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
CN = opensearch

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = opensearch
DNS.2 = opensearch-cluster-master
DNS.3 = opensearch-cluster-master.dav-monitoring
DNS.4 = opensearch-cluster-master.dav-monitoring.svc
DNS.5 = opensearch-cluster-master.dav-monitoring.svc.cluster.local
DNS.6 = localhost
IP.1 = 127.0.0.1
CONFEND

openssl genrsa -out root-ca-key.pem 2048
openssl req -new -x509 -sha256 -key root-ca-key.pem -out root-ca.pem -days 730 -subj "/CN=opensearch-ca"
openssl genrsa -out node-key.pem 2048
openssl req -new -key node-key.pem -out node.csr -config opensearch.cnf
openssl x509 -req -in node.csr -CA root-ca.pem -CAkey root-ca-key.pem -CAcreateserial -sha256 -out node.pem -days 730 -extensions req_ext -extfile opensearch.cnf

kubectl create secret generic $SECRET_NAME -n $NAMESPACE \
  --from-file=root-ca.pem=$CERT_DIR/root-ca.pem \
  --from-file=node.pem=$CERT_DIR/node.pem \
  --from-file=node-key.pem=$CERT_DIR/node-key.pem

kubectl get secret $SECRET_NAME -n $NAMESPACE
kubectl delete pod -l app=opensearch-cluster-master -n $NAMESPACE

EOF