variable "namespace" {
  type        = string
}

variable "credentials_name" {
  type        = string
}

variable "opensearch_host" {
  type        = string # opensearch-cluster-master
}

variable "opensearch_port" {
  type        = string
  default     = "9200"
}

variable "fluentbit_image_tag" {
  type        = string
  default     = "2.1.8"
}

variable "fluentbit_daemonset_yaml_path" {
  type        = string
  default     = "../../files/fluentbit_daemonset.yaml"
} 