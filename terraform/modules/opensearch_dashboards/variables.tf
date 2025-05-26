variable "namespace" {
  type        = string
}

variable "credentials_name" {
  type        = string
}

variable "opensearch_service_endpoint" {
  type        = string
}

variable "dashboards_helm_chart_version" {
  type        = string
  default     = "3.0.0" 
}

variable "dashboards_image_tag" {
  type        = string
  default     = "3.0.0" 
}

variable "dashboards_values_file" {
  type        = string
  default     = "../../files/opensearch_dashboards_values.yaml"
} 