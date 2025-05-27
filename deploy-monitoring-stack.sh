#!/bin/bash

kubectl create namespace dav-monitoring

mkdir -p opensearch-certs
cd opensearch-certs

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

cd ..

kubectl create secret generic opensearch-certs -n dav-monitoring \
  --from-file=root-ca.pem=opensearch-certs/root-ca.pem \
  --from-file=node.pem=opensearch-certs/node.pem \
  --from-file=node-key.pem=opensearch-certs/node-key.pem

kubectl create secret generic opensearch-credentials-secure -n dav-monitoring \
  --from-literal=username=admin \
  --from-literal=password=admin

kubectl label nodes --all role=opensearch-node

mkdir -p monitoring/opensearch/chart
cat > monitoring/opensearch/chart/values.yaml << 'VALUESEND'
replicas: 1
clusterName: opensearch
nodeGroup: master

image:
  repository: opensearchproject/opensearch
  imagePullPolicy: IfNotPresent
  tag: "3.0.0"

opensearchJavaOpts: "-Xms1g -Xmx1g"

config:
  opensearch.yml: |
    discovery.type: single-node
    network.host: 0.0.0.0
    
    plugins.security.disabled: false
    
    plugins.security.ssl.http.enabled: true
    plugins.security.ssl.transport.enabled: true
    
    plugins.security.ssl.http.pemcert_filepath: /usr/share/opensearch/config/certs/node.pem
    plugins.security.ssl.http.pemkey_filepath: /usr/share/opensearch/config/certs/node-key.pem
    plugins.security.ssl.http.pemtrustedcas_filepath: /usr/share/opensearch/config/certs/root-ca.pem
    plugins.security.ssl.transport.pemcert_filepath: /usr/share/opensearch/config/certs/node.pem
    plugins.security.ssl.transport.pemkey_filepath: /usr/share/opensearch/config/certs/node-key.pem
    plugins.security.ssl.transport.pemtrustedcas_filepath: /usr/share/opensearch/config/certs/root-ca.pem
    
    plugins.security.allow_default_init_securityindex: true
    
    plugins.query.datasources.encryption.masterkey: "A1B2C3D4E5F6G7H8"

resources:
  requests:
    cpu: "0.5"
    memory: "512Mi"
  limits:
    cpu: "2"
    memory: "2Gi"

nodeSelector:
  role: opensearch-node

persistence:
  enabled: true
  storageClass: "standard"
  size: 10Gi

singleNode: true

service:
  type: ClusterIP
  httpPort: 9200
  metricsPort: 9600

extraVolumes:
  - name: certs
    secret:
      secretName: opensearch-certs

extraVolumeMounts:
  - name: certs
    mountPath: /usr/share/opensearch/config/certs
    readOnly: true

extraEnvs:
  - name: OPENSEARCH_USERNAME
    valueFrom:
      secretKeyRef:
        name: opensearch-credentials-secure
        key: username
  - name: OPENSEARCH_PASSWORD
    valueFrom:
      secretKeyRef:
        name: opensearch-credentials-secure
        key: password

extraPlugins:
  - opensearch-performance-analyzer
  - opensearch-knn
  - opensearch-saml
  - opensearch-alerting
  - opensearch-anomaly-detection
  - opensearch-index-management
  - opensearch-sql
  - opensearch-prometheus
  - opensearch-machine-learning
VALUESEND

mkdir -p monitoring/opensearch-dashboards/chart
cat > monitoring/opensearch-dashboards/chart/values.yaml << 'DASHVALUESEND'
opensearchHosts: https://opensearch-cluster-master:9200

image:
  repository: opensearchproject/opensearch-dashboards
  tag: "3.0.0"
  pullPolicy: IfNotPresent

opensearchAccount:
  secret: opensearch-credentials-secure
  keyPassphrase:
    enabled: false

config:
  opensearch_dashboards.yml: |
    server.name: opensearch-dashboards
    server.host: "0.0.0.0"
    opensearch.hosts: ["https://opensearch-cluster-master:9200"]
    opensearch.ssl.verificationMode: none
    opensearch.username: ${OPENSEARCH_USERNAME}
    opensearch.password: ${OPENSEARCH_PASSWORD}
    opensearch.requestHeadersAllowlist: ["Authorization", "X-Security-Tenant", "securitytenant"]

service:
  type: ClusterIP
  port: 5601
  targetPort: 5601
DASHVALUESEND

mkdir -p monitoring/fluentbit
cat > monitoring/fluentbit/fluentbit-daemonset.yaml << 'FLUENTVALUESEND'
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluentbit
  namespace: dav-monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: fluentbit
rules:
  - apiGroups: [""]
    resources:
      - namespaces
      - pods
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: fluentbit
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: fluentbit
subjects:
  - kind: ServiceAccount
    name: fluentbit
    namespace: dav-monitoring
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentbit-config
  namespace: dav-monitoring
  labels:
    app.kubernetes.io/name: fluentbit
data:
  fluent-bit.conf: |-
    [SERVICE]
        Flush         1
        Daemon        Off
        Log_Level     info
        Parsers_File  parsers.conf
        HTTP_Server   On
        HTTP_Listen   0.0.0.0
        HTTP_Port     2020

    @INCLUDE inputs.conf
    @INCLUDE filters.conf
    @INCLUDE outputs.conf

  inputs.conf: |-
    [INPUT]
        Name              tail
        Tag               kube.*
        Path              /var/log/containers/*.log
        Parser            cri
        DB                /var/log/flb_kube.db
        Mem_Buf_Limit     5MB
        Skip_Long_Lines   On
        Refresh_Interval  10

  filters.conf: |-
    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL            https://kubernetes.default.svc:443
        Kube_CA_File        /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        Kube_Token_File     /var/run/secrets/kubernetes.io/serviceaccount/token
        Kube_Tag_Prefix     kube.
        Merge_Log           On
        K8S-Logging.Parser  On
        K8S-Logging.Exclude Off

    [FILTER]
        Name                rewrite_tag
        Match               kube.*
        Rule                $kubernetes['namespace_name'] ^default$ $kubernetes['container_name'] ^expressops(.*)$ expressops.logs true
        Emitter_Name        re_emitted

  outputs.conf: |-
    [OUTPUT]
        Name            opensearch
        Match           expressops.logs
        Host            opensearch-standard
        Port            9200
        HTTP_User       ${OPENSEARCH_USERNAME}
        HTTP_Passwd     ${OPENSEARCH_PASSWORD}
        Index           expressops
        Suppress_Type_Name On
        tls             On
        tls.verify      Off

    [OUTPUT]
        Name            opensearch
        Match           *
        Host            opensearch-standard
        Port            9200
        HTTP_User       ${OPENSEARCH_USERNAME}
        HTTP_Passwd     ${OPENSEARCH_PASSWORD}
        Index           logs

  parsers.conf: |-
    [PARSER]
        Name   cri
        Format regex
        Regex  ^(?<time>[^ ]+) (?<stream>stdout|stderr) (?<logtag>[^ ]*) (?<log>.*)$
        Time_Key    time
        Time_Format %Y-%m-%dT%H:%M:%S.%L%z

---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentbit
  namespace: dav-monitoring
  labels:
    app.kubernetes.io/name: fluentbit
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: fluentbit
  template:
    metadata:
      labels:
        app.kubernetes.io/name: fluentbit
    spec:
      serviceAccountName: fluentbit
      containers:
      - name: fluentbit
        image: fluent/fluent-bit:2.1.8
        imagePullPolicy: IfNotPresent
        env:
          - name: OPENSEARCH_USERNAME
            valueFrom:
              secretKeyRef:
                name: opensearch-credentials-secure
                key: username
          - name: OPENSEARCH_PASSWORD
            valueFrom:
              secretKeyRef:
                name: opensearch-credentials-secure
                key: password
        volumeMounts:
        - name: config
          mountPath: /fluent-bit/etc/
        - name: varlog
          mountPath: /var/log
        - name: varlibcontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        - name: etcmachineid
          mountPath: /etc/machine-id
          readOnly: true
        ports:
        - name: http
          containerPort: 2020
      volumes:
      - name: config
        configMap:
          name: fluentbit-config
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibcontainers
        hostPath:
          path: /var/lib/docker/containers
      - name: etcmachineid
        hostPath:
          path: /etc/machine-id
          type: File
FLUENTVALUESEND

cat > install-monitoring.sh << 'INSTALLSCRIPTEND'
#!/bin/bash

helm repo add opensearch https://opensearch-project.github.io/helm-charts/
helm repo update

helm install opensearch opensearch/opensearch -n dav-monitoring -f monitoring/opensearch/chart/values.yaml

kubectl wait --for=condition=ready pod -l app=opensearch-cluster-master -n dav-monitoring --timeout=300s

helm install opensearch-dashboards opensearch/opensearch-dashboards -n dav-monitoring -f monitoring/opensearch-dashboards/chart/values.yaml

kubectl apply -f monitoring/fluentbit/fluentbit-daemonset.yaml
INSTALLSCRIPTEND

chmod +x install-monitoring.sh
