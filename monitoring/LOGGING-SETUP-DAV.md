# GUÍA INSTALACIÓN

## Orden de instalación
1. OpenSearch → 2. Dashboards → 3. Logstash → 4. Filebeat

## Comandos

### 1. Instalar 
```bash
helm install my-release oci://registry-1.docker.io/bitnamicharts/opensearch \
  --namespace dav-monitoring

helm install opensearch-dashboards oci://registry-1.docker.io/bitnamicharts/opensearch-dashboards \
  --namespace dav-monitoring \
  --set opensearch.hosts[0]="http://my-release-opensearch.dav-monitoring.svc.cluster.local:9200"

helm install logstash bitnami/logstash \
  --namespace dav-monitoring \
  --set logstashConfig.logstash.conf="input { beats { port => 5044 } } output { opensearch { hosts => [\"http://my-release-opensearch:9200\"] index => \"logstash-index\" } }"

kubectl apply -f monitoring/filebeat/filebeat.yaml                                                 
kubectl apply -f monitoring/filebeat/filebeat-config.yaml

```

### 2.  CHECKEA
1. Ve a "Stack Management" > "Index Patterns"
2. Crea patrón `expressops-logs-*`
3. Selecciona `@timestamp`