variable "namespace" {
  description = "Kubernetes namespace for Fluent Bit deployment."
  type        = string
}

variable "credentials_name" {
  description = "Name of the Kubernetes secret for OpenSearch admin credentials used by Fluent Bit."
  type        = string
}

variable "opensearch_host" {
  description = "OpenSearch host for Fluent Bit output."
  type        = string # opensearch-cluster-master
}

variable "opensearch_port" {
  description = "OpenSearch port for Fluent Bit output."
  type        = string
  default     = "9200"
}

variable "fluentbit_image_tag" {
  description = "Docker image tag for Fluent Bit."
  type        = string
  default     = "2.1.8"
}

variable "fluentbit_daemonset_yaml_path" {
  description = "Path to the Fluent Bit DaemonSet YAML manifest (if using kubectl_manifest or similar)."
  type        = string
  default     = "../../files/fluentbit_daemonset.yaml"
} 