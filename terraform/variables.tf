variable "slack_webhook_url" {
  type        = string
  sensitive   = true
}

variable "service_account_email" {
  type        = string
  default     = "expressops-external-secrets@fc-it-school-2025.iam.gserviceaccount.com"  
}

variable "deploy_k8s_resources" {
  type        = bool
  default     = false
}

variable "k8s_namespace" {
  type        = string
  default     = "default" # expressops-dev in case of the other version of the app in rama_nacho
}

variable "namespace" {
  type        = string
  default     = "dav-monitoring"
}

variable "opensearch_certs_path" {
  type        = string
  default     = "./opensearch-certs-generated"
}

variable "opensearch_admin_username" {
  type        = string
  default     = "admin"
  sensitive   = true
}

variable "opensearch_admin_password" {
  type        = string
  default     = "admin"
  sensitive   = true
} 