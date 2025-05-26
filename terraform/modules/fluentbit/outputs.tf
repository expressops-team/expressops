output "fluentbit_daemonset_name" {
  description = "Name of the Fluent Bit DaemonSet."
  value       = kubernetes_daemonset.fluentbit.metadata[0].name
}

output "fluentbit_service_account_name" {
  description = "Name of the Fluent Bit Service Account."
  value       = kubernetes_service_account.fluentbit.metadata[0].name
} 