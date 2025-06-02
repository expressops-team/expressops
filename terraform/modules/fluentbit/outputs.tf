output "fluentbit_daemonset_name" {
  value       = kubernetes_daemonset.fluentbit.metadata[0].name
}

output "fluentbit_service_account_name" {
  value       = kubernetes_service_account.fluentbit.metadata[0].name
} 