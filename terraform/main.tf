provider "google" {
  project = "fc-it-school-2025"
  region  = "europe-west1"
}

resource "google_secret_manager_secret" "slack_webhook" {
  secret_id = "slack-webhook"
  
  replication {
    user_managed {
      replicas {
        location = "europe-west1"
      }
    }
  }

  labels = {
    environment = "school"
    app         = "expressops"
    managed_by  = "terraform"
  }
}

resource "google_secret_manager_secret_version" "slack_webhook_version" {
  secret      = google_secret_manager_secret.slack_webhook.id
  secret_data = var.slack_webhook_url
}

resource "google_secret_manager_secret_iam_member" "secret_accessor" {
  secret_id  = google_secret_manager_secret.slack_webhook.id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${var.service_account_email}"
}

resource "kubernetes_namespace" "monitoring_namespace" {
  metadata {
    name = var.namespace
  }
}

module "kubernetes_secrets" {
  source = "./modules/kubernetes_secrets"

  namespace                 = kubernetes_namespace.monitoring_namespace.metadata[0].name
  opensearch_certs_path     = var.opensearch_certs_path
  opensearch_admin_username = var.opensearch_admin_username
  opensearch_admin_password = var.opensearch_admin_password
  
  depends_on = [kubernetes_namespace.monitoring_namespace]
}

module "opensearch" {
  source = "./modules/opensearch"

  namespace        = kubernetes_namespace.monitoring_namespace.metadata[0].name
  secrets_name     = module.kubernetes_secrets.opensearch_certs_secret_name
  credentials_name = module.kubernetes_secrets.opensearch_credentials_secret_name
  
  depends_on = [module.kubernetes_secrets]
}

module "opensearch_dashboards" {
  source = "./modules/opensearch_dashboards"

  namespace                   = kubernetes_namespace.monitoring_namespace.metadata[0].name
  credentials_name            = module.kubernetes_secrets.opensearch_credentials_secret_name
  opensearch_service_endpoint = module.opensearch.opensearch_url

  depends_on = [module.opensearch]
}

module "fluentbit" {
  source = "./modules/fluentbit"

  namespace        = kubernetes_namespace.monitoring_namespace.metadata[0].name
  credentials_name = module.kubernetes_secrets.opensearch_credentials_secret_name
  opensearch_host  = module.opensearch.opensearch_service_name

  depends_on = [module.opensearch]
}