resource "helm_release" "opensearch_dashboards" {
  name       = "opensearch-dashboards"
  repository = "https://opensearch-project.github.io/helm-charts"
  chart      = "opensearch-dashboards"
  version    = var.dashboards_helm_chart_version
  namespace  = var.namespace

  values = [
    templatefile(var.dashboards_values_file, {
      dashboards_image_tag      = var.dashboards_image_tag,
      opensearch_hosts          = var.opensearch_service_endpoint,
      opensearch_secret_name    = var.credentials_name 
    })
  ]
} 