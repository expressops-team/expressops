terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.10"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.20"
    }
    local = {
      source = "hashicorp/local"
      version = "~> 2.4.0"
    }
    tls = {
      source = "hashicorp/tls"
      version = "~> 4.0"
    }
    google = {
      source = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "kubernetes" {
}

provider "helm" {
  kubernetes {
  }
} 