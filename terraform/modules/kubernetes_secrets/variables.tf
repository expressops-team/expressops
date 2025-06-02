variable "namespace" {
  type        = string
}

variable "opensearch_certs_path" {
  type        = string
}

variable "opensearch_admin_username" {
  type        = string
  sensitive   = true
}

variable "opensearch_admin_password" {
  type        = string
  sensitive   = true
} 