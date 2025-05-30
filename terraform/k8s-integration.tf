
# Define SecretStore
resource "kubernetes_manifest" "gcp_secret_store" {
  count = var.deploy_k8s_resources ? 1 : 0
  
  manifest = {
    apiVersion = "external-secrets.io/v1beta1"
    kind       = "SecretStore"
    metadata = {
      name      = "gcp-secretstore"
      namespace = var.k8s_namespace
    }
    spec = {
      provider = {
        gcpsm = {
          projectID = "fc-it-school-2025"
          auth = {
            secretRef = {
              secretAccessKeySecretRef = {
                name      = "gcpsm-credentials" 
                namespace = var.k8s_namespace
                key       = "service-account.json"
              }
            }
          }
        }
      }
    }
  }
}

# Define  ExternalSecret 
resource "kubernetes_manifest" "slack_webhook_external_secret" {
  count = var.deploy_k8s_resources ? 1 : 0
  depends_on = [kubernetes_manifest.gcp_secret_store]
  
  manifest = {
    apiVersion = "external-secrets.io/v1beta1"
    kind       = "ExternalSecret"
    metadata = {
      name      = "expressops-slack-external-secret"
      namespace = var.k8s_namespace
    }
    spec = {
      refreshInterval = "1h"
      secretStoreRef = {
        name = "gcp-secretstore"
        kind = "SecretStore"
      }
      target = {
        name           = "expressops-slack-secret"
        creationPolicy = "Owner"
      }
      data = [
        {
          secretKey = "SLACK_WEBHOOK_URL"
          remoteRef = {
            key = "projects/fc-it-school-2025/secrets/slack-webhook"
          }
        }
      ]
    }
  }
} 