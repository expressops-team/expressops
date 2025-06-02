# fluentbit-daemonset.yaml decomposed into Terraform resources

resource "kubernetes_service_account" "fluentbit" {
  metadata {
    name      = "fluentbit"
    namespace = var.namespace
  }
}

resource "kubernetes_cluster_role" "fluentbit" {
  metadata {
    name = "fluentbit-${var.namespace}"
  }
  rule {
    api_groups = [""]
    resources  = ["namespaces", "pods"]
    verbs      = ["get", "list", "watch"]
  }
}

resource "kubernetes_cluster_role_binding" "fluentbit" {
  metadata {
    name = "fluentbit-${var.namespace}" # Ensure unique name
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role.fluentbit.metadata[0].name
  }
  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.fluentbit.metadata[0].name
    namespace = var.namespace
  }
}

resource "kubernetes_config_map" "fluentbit_config" {
  metadata {
    name      = "fluentbit-config"
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name" = "fluentbit"
    }
  }
  data = {
    "fluent-bit.conf" = <<-EOT
    [SERVICE]
        Flush         1
        Daemon        Off
        Log_Level     info
        Parsers_File  parsers.conf
        HTTP_Server   On
        HTTP_Listen   0.0.0.0
        HTTP_Port     2020

    @INCLUDE inputs.conf
    @INCLUDE filters.conf
    @INCLUDE outputs.conf
    EOT

    "inputs.conf" = <<-EOT
    [INPUT]
        Name              tail
        Tag               kube.*
        Path              /var/log/containers/*.log
        Parser            cri
        DB                /var/log/flb_kube.db
        Mem_Buf_Limit     5MB
        Skip_Long_Lines   On
        Refresh_Interval  10
    EOT

    "filters.conf" = <<-EOT
    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL            https://kubernetes.default.svc:443
        Kube_CA_File        /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        Kube_Token_File     /var/run/secrets/kubernetes.io/serviceaccount/token
        Kube_Tag_Prefix     kube.
        Merge_Log           On
        K8S-Logging.Parser  On
        K8S-Logging.Exclude Off
    EOT

    "outputs.conf" = <<-EOT
    [OUTPUT]
        Name            opensearch
        Match           *
        Host            ${var.opensearch_host}
        Port            ${var.opensearch_port}
        HTTP_User       $${OPENSEARCH_USERNAME} # Escaped for env var interpolation by Fluent Bit
        HTTP_Passwd     $${OPENSEARCH_PASSWORD} # Escaped for env var interpolation by Fluent Bit
        Index           logs # Or make this configurable
        Suppress_Type_Name On
        tls             On
        tls.verify      Off # Consider true for production with proper CA
    EOT

    "parsers.conf" = <<-EOT
    [PARSER]
        Name   cri
        Format regex
        Regex  ^(?<time>[^ ]+) (?<stream>stdout|stderr) (?<logtag>[^ ]*) (?<log>.*)$
        Time_Key    time
        Time_Format %Y-%m-%dT%H:%M:%S.%L%z
    EOT
  }
}

resource "kubernetes_daemonset" "fluentbit" {
  metadata {
    name      = "fluentbit"
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name" = "fluentbit"
    }
  }
  spec {
    selector {
      match_labels = {
        "app.kubernetes.io/name" = "fluentbit"
      }
    }
    template {
      metadata {
        labels = {
          "app.kubernetes.io/name" = "fluentbit"
        }
      }
      spec {
        service_account_name = kubernetes_service_account.fluentbit.metadata[0].name
        container {
          name  = "fluentbit"
          image = "fluent/fluent-bit:${var.fluentbit_image_tag}"
          image_pull_policy = "IfNotPresent"

          env {
            name = "OPENSEARCH_USERNAME"
            value_from {
              secret_key_ref {
                name = var.credentials_name
                key  = "username"
              }
            }
          }
          env {
            name = "OPENSEARCH_PASSWORD"
            value_from {
              secret_key_ref {
                name = var.credentials_name
                key  = "password"
              }
            }
          }

          port {
            name           = "http"
            container_port = 2020
          }

          volume_mount {
            name       = "config"
            mount_path = "/fluent-bit/etc/"
          }
          volume_mount {
            name       = "varlog"
            mount_path = "/var/log"
          }
          volume_mount {
            name       = "varlibcontainers"
            mount_path = "/var/lib/docker/containers"
            read_only  = true
          }
          volume_mount {
            name       = "etcmachineid"
            mount_path = "/etc/machine-id"
            read_only  = true
          }
        }

        volume {
          name = "config"
          config_map {
            name = kubernetes_config_map.fluentbit_config.metadata[0].name
          }
        }
        volume {
          name = "varlog"
          host_path {
            path = "/var/log"
          }
        }
        volume {
          name = "varlibcontainers"
          host_path {
            path = "/var/lib/docker/containers"
          }
        }
        volume {
          name = "etcmachineid"
          host_path {
            path = "/etc/machine-id"
            type = "File"
          }
        }
      }
    }
  }
  depends_on = [
    kubernetes_service_account.fluentbit,
    kubernetes_cluster_role_binding.fluentbit,
    kubernetes_config_map.fluentbit_config
  ]
} 