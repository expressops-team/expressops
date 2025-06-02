resource "helm_release" "opensearch" {
  name       = "opensearch"
  repository = "https://opensearch-project.github.io/helm-charts"
  chart      = "opensearch"
  version    = var.opensearch_helm_chart_version
  namespace  = var.namespace

  values = [
  templatefile(abspath("${path.module}/../../files/opensearch_values.yaml"), {
      opensearch_image_tag = var.opensearch_image_tag,
      secrets_name         = var.secrets_name,
      credentials_name     = var.credentials_name
    })
  ]

  depends_on = [
  ]
} 