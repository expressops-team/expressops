# not real values, CHANGE THEM BEFORE USING

slack_webhook_url     = "https://hooks.slack.com/services/TU/SLACK/WEBHOOK"
service_account_email = "tu-sa@tu-proyecto-gcp.iam.gserviceaccount.com"

deploy_k8s_resources = false
k8s_namespace        = "default"

namespace                 = "dav-monitoring" 
opensearch_certs_path     = "./opensearch-certs-generated" 
opensearch_admin_username = "admin" 
opensearch_admin_password = "ExpressOps123" # PUT YOUR PASSWORD HERE or pass it as a secret