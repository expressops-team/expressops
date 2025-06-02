output "secret_name" {
  value       = google_secret_manager_secret.slack_webhook.name
}

output "secret_id" {
  value       = google_secret_manager_secret.slack_webhook.id
}

output "secret_version" {
  value       = google_secret_manager_secret_version.slack_webhook_version.version
}

output "gcp_secret_reference" {
  value       = "projects/fc-it-school-2025/secrets/slack-webhook/versions/latest"
  description = "Reference ==> secret for use with External Secrets Operator"
}

output "opensearch_dashboards_url" {
  value       = module.opensearch_dashboards.opensearch_dashboards_url
}

output "opensearch_url" {
  value       = module.opensearch.opensearch_url
} 