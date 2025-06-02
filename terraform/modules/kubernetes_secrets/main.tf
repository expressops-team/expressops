resource "null_resource" "generate_opensearch_certs" {
  triggers = {
    cert_generation_trigger = var.namespace 
  }

  provisioner "local-exec" {
    command = "bash ${path.module}/../../files/generate_certs.sh ${var.opensearch_certs_path} opensearch ${var.namespace}"
  }
}

resource "kubernetes_secret" "opensearch_certs" {
  depends_on = [null_resource.generate_opensearch_certs]
  metadata {
    name      = "opensearch-certs"
    namespace = var.namespace
  }

  data = {
    "root-ca.pem" = fileexists("${var.opensearch_certs_path}/root-ca.pem") ? file("${var.opensearch_certs_path}/root-ca.pem") : ""
    "node.pem"    = fileexists("${var.opensearch_certs_path}/node.pem") ? file("${var.opensearch_certs_path}/node.pem") : ""
    "node-key.pem"= fileexists("${var.opensearch_certs_path}/node-key.pem") ? file("${var.opensearch_certs_path}/node-key.pem") : ""
  }

  type = "Opaque"
}

resource "kubernetes_secret" "opensearch_credentials" {
  metadata {
    name      = "opensearch-credentials-secure"
    namespace = var.namespace
  }

  data = {
    username = var.opensearch_admin_username
    password = var.opensearch_admin_password
  }

  type = "Opaque"
} 