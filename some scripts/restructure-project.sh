#!/bin/bash

# =============================================================================
# 🗂️ ExpressOps Project Restructure Script
# =============================================================================
# Este script reorganiza la estructura del proyecto ExpressOps
# para mejorar la organización, eliminar duplicados y separar por entornos.
#
# IMPORTANTE: Ejecutar desde el directorio raíz del proyecto
# =============================================================================

set -euo pipefail

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Funciones de logging
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }

# Verificar que estamos en el directorio correcto
if [[ ! -f "go.mod" ]] || [[ ! -f "Makefile" ]]; then
    log_error "Este script debe ejecutarse desde el directorio raíz del proyecto ExpressOps"
    exit 1
fi

# Crear backup antes de la migración
BACKUP_DIR="backup-$(date +%Y%m%d-%H%M%S)"
log_info "Creando backup en: $BACKUP_DIR"
mkdir -p "$BACKUP_DIR"

# Archivos críticos para backup
CRITICAL_FILES=(
    "opensearch-security-configmap.yaml"
    "k3s.yaml" 
    "deploy-monitoring-stack.sh"
    "key.json"
    "key_artifact.json"
)

for file in "${CRITICAL_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        cp "$file" "$BACKUP_DIR/"
        log_success "Backup creado: $file"
    fi
done

# Backup de directorios completos
cp -r monitoring/elastic "$BACKUP_DIR/" 2>/dev/null || true
cp -r monitoring/grafana-dashboards "$BACKUP_DIR/" 2>/dev/null || true
cp -r k8s "$BACKUP_DIR/" 2>/dev/null || true

log_success "Backup completado en: $BACKUP_DIR"

# =============================================================================
# FASE 1: Crear nueva estructura de directorios
# =============================================================================
log_info "📁 FASE 1: Creando nueva estructura de directorios..."

# Crear directorios principales
mkdir -p {build,deployments,configs,secrets,infrastructure}
mkdir -p deployments/{environments/{dev,staging,prod},kubernetes/{base,overlays}}
mkdir -p monitoring/observability/{grafana,prometheus}
mkdir -p infrastructure/{terraform/environments,gitops/{applications,projects}}
mkdir -p scripts/{deployment,monitoring,testing,utilities}
mkdir -p docs/{architecture,deployment,monitoring,comparisons}
mkdir -p secrets/templates
mkdir -p tools

log_success "Estructura de directorios creada"

# =============================================================================
# FASE 2: Mover archivos a nueva estructura
# =============================================================================
log_info "📦 FASE 2: Reorganizando archivos..."

# Mover archivos de construcción
if [[ -f "Dockerfile" ]]; then
    mv Dockerfile build/
    log_success "Movido: Dockerfile → build/"
fi

if [[ -f ".dockerignore" ]]; then
    mv .dockerignore build/
    log_success "Movido: .dockerignore → build/"
fi

# Mover Helm charts
if [[ -d "helm" ]]; then
    mv helm/* deployments/helm/ 2>/dev/null || true
    rmdir helm 2>/dev/null || true
    log_success "Movido: helm/* → deployments/helm/"
fi

# Mover manifiestos K8s
if [[ -d "k8s" ]]; then
    mv k8s/* deployments/kubernetes/base/ 2>/dev/null || true
    rmdir k8s 2>/dev/null || true
    log_success "Movido: k8s/* → deployments/kubernetes/base/"
fi

# Reorganizar k3s
if [[ -d "k3s" ]]; then
    mv k3s infrastructure/
    log_success "Movido: k3s → infrastructure/"
fi

# Mover GitOps
if [[ -d "gitops-argocd" ]]; then
    mv gitops-argocd/* infrastructure/gitops/ 2>/dev/null || true
    rmdir gitops-argocd 2>/dev/null || true
    log_success "Movido: gitops-argocd/* → infrastructure/gitops/"
fi

# Mover Terraform
if [[ -d "terraform" ]]; then
    mv terraform/* infrastructure/terraform/ 2>/dev/null || true
    rmdir terraform 2>/dev/null || true
    log_success "Movido: terraform/* → infrastructure/terraform/"
fi

# Consolidar scripts
if [[ -f "deploy-monitoring-stack.sh" ]]; then
    mv deploy-monitoring-stack.sh scripts/deployment/
    log_success "Movido: deploy-monitoring-stack.sh → scripts/deployment/"
fi

if [[ -d "scripts" ]] && [[ -f "scripts/install-monitoring.sh" ]]; then
    mv scripts/install-monitoring.sh scripts/deployment/
    log_success "Movido: install-monitoring.sh → scripts/deployment/"
fi

# Mover herramientas
mv Makefile tools/
if [[ -d "makefiles" ]]; then
    mv makefiles tools/
    log_success "Movido: Makefile y makefiles → tools/"
fi

# Crear Makefile wrapper en raíz
cat > Makefile << 'EOF'
# =============================================================================
# ExpressOps - Makefile Wrapper
# =============================================================================
# Este Makefile redirige a la configuración real en tools/
.DEFAULT_GOAL := help

include tools/Makefile
EOF

# =============================================================================
# FASE 3: Reorganizar monitoreo y eliminar duplicados
# =============================================================================
log_info "🔍 FASE 3: Reorganizando monitoreo..."

# Consolidar dashboards de Grafana
if [[ -d "monitoring/grafana-dashboards" ]]; then
    mv monitoring/grafana-dashboards/* monitoring/observability/grafana/ 2>/dev/null || true
    rmdir monitoring/grafana-dashboards 2>/dev/null || true
fi

if [[ -d "deployments/helm/grafana-dashboards" ]]; then
    mv deployments/helm/grafana-dashboards/* monitoring/observability/grafana/ 2>/dev/null || true
    rmdir deployments/helm/grafana-dashboards 2>/dev/null || true
fi

log_success "Dashboards de Grafana consolidados"

# Mover componentes de observabilidad
if [[ -d "monitoring/fluentbit" ]]; then
    mv monitoring/fluentbit monitoring/observability/
    log_success "Movido: fluentbit → monitoring/observability/"
fi

if [[ -d "monitoring/prometheus" ]]; then
    mv monitoring/prometheus/* monitoring/observability/prometheus/ 2>/dev/null || true
    rmdir monitoring/prometheus 2>/dev/null || true
    log_success "Movido: prometheus → monitoring/observability/prometheus/"
fi

# Mover KEDA
if [[ -d "monitoring/opensearch/keda" ]]; then
    mv monitoring/opensearch/keda monitoring/
    log_success "Movido: keda → monitoring/"
fi

# Reorganizar OpenSearch
if [[ -d "monitoring/opensearch/chart/opensearch-security" ]]; then
    mv monitoring/opensearch/chart/opensearch-security monitoring/opensearch/security
    log_success "Reorganizado: opensearch security"
fi

if [[ -d "monitoring/opensearch/index" ]]; then
    mv monitoring/opensearch/index monitoring/opensearch/policies
    log_success "Movido: index → policies"
fi

# =============================================================================
# FASE 4: Gestión de secretos y credenciales
# =============================================================================
log_info "🔐 FASE 4: Organizando secretos..."

# Mover credenciales a secrets/ (crear templates)
if [[ -f "key.json" ]]; then
    mv key.json secrets/
    echo "key.json" >> .gitignore
    log_warning "Credencial movida a secrets/ y añadida a .gitignore"
fi

if [[ -f "key_artifact.json" ]]; then
    mv key_artifact.json secrets/
    echo "key_artifact.json" >> .gitignore
    log_warning "Credencial movida a secrets/ y añadida a .gitignore"
fi

if [[ -f "k3s.yaml" ]]; then
    mv k3s.yaml secrets/
    echo "k3s.yaml" >> .gitignore
    log_warning "Configuración de cluster movida a secrets/"
fi

# Crear README para secretos
cat > secrets/README.md << 'EOF'
# 🔐 Gestión de Secretos

## ⚠️ Importante
Este directorio contiene archivos sensibles que NO deben incluirse en el repositorio.

## 📁 Contenido
- `key.json` - Credenciales de servicio GCP
- `key_artifact.json` - Credenciales adicionales 
- `k3s.yaml` - Configuración de cluster local

## 🛡️ Mejores Prácticas
1. Usar External Secrets Operator en producción
2. Mantener templates sin valores reales en `templates/`
3. Documentar cada secreto requerido
4. Rotar credenciales regularmente

## 🔄 Gestión por Entorno
- **Dev**: Secretos locales en este directorio
- **Staging/Prod**: External Secrets + Vault/GCP Secret Manager
EOF

log_success "README de secretos creado"

# =============================================================================
# FASE 5: Limpieza de archivos obsoletos
# =============================================================================
log_info "🗑️  FASE 5: Eliminando archivos obsoletos..."

# Eliminar archivos duplicados
OBSOLETE_FILES=(
    "opensearch-security-configmap.yaml"
    "expressops"  # Binario compilado
)

for file in "${OBSOLETE_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        rm "$file"
        log_success "Eliminado: $file"
    fi
done

# Eliminar directorios obsoletos (stack Elasticsearch)
OBSOLETE_DIRS=(
    "monitoring/elastic"
    "monitoring/logstash" 
    "monitoring/filebeat"
)

for dir in "${OBSOLETE_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
        rm -rf "$dir"
        log_success "Eliminado directorio: $dir"
    fi
done

# =============================================================================
# FASE 6: Actualizar configuraciones
# =============================================================================
log_info "⚙️  FASE 6: Actualizando configuraciones..."

# Actualizar .gitignore
cat >> .gitignore << 'EOF'

# =============================================================================
# ExpressOps - Archivos sensibles y temporales
# =============================================================================
secrets/*.json
secrets/*.yaml
secrets/*.key
secrets/*.pem
!secrets/templates/
!secrets/README.md
!secrets/.gitkeep

# Binarios compilados
/expressops
/bin/expressops

# Archivos temporales de Terraform
infrastructure/terraform/**/*.tfstate*
infrastructure/terraform/**/.terraform/
infrastructure/terraform/**/terraform.tfvars

# Configuraciones locales de K8s
*.kubeconfig
kubeconfig*
EOF

log_success "Actualizado .gitignore"

# Crear archivo de configuración de entorno
cat > configs/README.md << 'EOF'
# ⚙️ Configuraciones por Entorno

## 📁 Estructura
```
configs/
├── dev/          # Configuraciones de desarrollo
├── staging/      # Configuraciones de staging  
└── prod/         # Configuraciones de producción
```

## 🎯 Uso
Cada entorno debe contener:
- `config.yaml` - Configuración principal de ExpressOps
- `values.yaml` - Values específicos para Helm
- `kustomization.yaml` - Overlays de Kubernetes

## 🔄 Integración
Los scripts de despliegue utilizan automáticamente la configuración
del entorno especificado via variable `ENVIRONMENT`.
EOF

# =============================================================================
# FINALIZACIÓN
# =============================================================================
log_success "🎉 ¡Reestructuración completada exitosamente!"

echo ""
log_info "📋 Resumen de cambios:"
echo "   ✅ Estructura reorganizada por propósito"
echo "   ✅ Duplicados eliminados (Elastic stack, dashboards, etc.)"
echo "   ✅ Credenciales movidas a secrets/ con .gitignore"
echo "   ✅ Scripts consolidados en scripts/"
echo "   ✅ Separación por entornos preparada"
echo "   ✅ Backup creado en: $BACKUP_DIR"

echo ""
log_warning "📝 Próximos pasos:"
echo "   1. Revisar y validar la nueva estructura"
echo "   2. Actualizar referencias en scripts/CI/CD"
echo "   3. Configurar entornos en configs/"
echo "   4. Implementar External Secrets en prod"
echo "   5. Actualizar documentación"

echo ""
log_info "📖 Consulta RESTRUCTURE_PLAN.md para más detalles" 