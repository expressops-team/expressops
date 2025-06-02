variable "namespace" {
  type        = string
}

variable "secrets_name" {
  type        = string
}

variable "credentials_name" {
  type        = string
}

variable "opensearch_helm_chart_version" {
  type        = string
  default     = "3.0.0" 
}

variable "opensearch_image_tag" {
  type        = string
  default     = "3.0.0" 
}

variable "opensearch_values_file" {
  type        = string
  default     = "../../files/opensearch_values.yaml"
} 