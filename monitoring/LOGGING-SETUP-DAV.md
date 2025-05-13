# GUÍA INSTALACIÓN

## Orden de instalación
1. OpenSearch → 2. Dashboards → 3. Logstash → 4. Filebeat

## Comandos

### 1. Instalar Logstash
```bash
helm repo add elastic https://helm.elastic.co && helm repo update
helm install logstash elastic/logstash -n dav-monitoring -f monitoring/logstash/logstash-values.yaml
```

### 2. Instalar Filebeat
```bash
helm install filebeat elastic/filebeat -n dav-monitoring -f monitoring/filebeat/filebeat-values.yaml
```

### 3. Verificación
```bash
kubectl get pods -n dav-monitoring
kubectl port-forward -n dav-monitoring svc/opensearch-dashboards 5601:5601
```

### 4. En OpenSearch Dashboards CHECKEA
1. Ve a "Stack Management" > "Index Patterns"
2. Crea patrón `expressops-logs-*`
3. Selecciona `@timestamp`