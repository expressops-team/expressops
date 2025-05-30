# Vendor Charts

Este directorio contiene los "wrapper charts" o "charts paraguas" para gestionar charts de terceros con valores personalizados. Esta estructura permite una gestión limpia y organizada de dependencias externas.

## Estructura

```
vendor-charts/
├── juicefs-csi-driver/       # Wrapper para el chart de JuiceFS CSI Driver
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
├── argo-cd/                  # Wrapper para el chart de Argo CD
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
├── prometheus-stack/         # Wrapper para el chart de Prometheus Stack
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
├── loki/                     # Wrapper para el chart de Loki
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
├── fluentbit/                # Wrapper para el chart de Fluent Bit
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
├── grafana/                  # Wrapper para el chart de Grafana
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
├── external-secrets/         # Wrapper para el chart de External Secrets Operator
│   ├── Chart.yaml            # Define la dependencia del chart oficial
│   └── values.yaml           # Valores personalizados para el chart
└── README.md                 # Este archivo
```

## Uso

### Instalar dependencias

Antes de instalar cualquier chart, es necesario actualizar las dependencias:

```bash
cd k3s
make update-dependencies
```

O manualmente:

```bash
helm dependency update vendor-charts/<nombre-del-chart>
```

### Instalar componentes individuales

#### JuiceFS CSI Driver

```bash
cd k3s
make install-juicefs
```

#### Argo CD

```bash
cd k3s
make install-argocd
```

#### Prometheus Stack

```bash
cd k3s
make install-prometheus
```

#### Loki

```bash
cd k3s
make install-loki
```

#### Fluent Bit

```bash
cd k3s
make install-fluentbit
```

#### Grafana

```bash
cd k3s
make install-grafana
```

#### External Secrets Operator

```bash
cd k3s
make install-external-secrets
```

### Instalar stack completo de monitorización

```bash
cd k3s
make install-monitoring-stack
```

## Actualizar valores

Para modificar la configuración de los charts, edita los archivos `values.yaml` en el directorio correspondiente. Luego, actualiza la instalación usando los comandos anteriores.

## Versiones

Para actualizar la versión de un chart dependiente, edita el campo `version` en el archivo `Chart.yaml` correspondiente. 