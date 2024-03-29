data "octopusdeploy_machine_policies" "default_machine_policy" {
  ids          = null
  partial_name = "Default Machine Policy"
  skip         = 0
  take         = 1
}

resource octopusdeploy_kubernetes_cluster_deployment_target test_k8s {
  cluster_url                       = "https://cluster"
  environments                      = [octopusdeploy_environment.test_environment.id]
  name                              = "Test"
  roles                             = ["eks"]
  cluster_certificate               = ""
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  namespace                         = ""
  skip_tls_verification             = true
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
  uri                               = ""

  container {
    feed_id = ""
    image   = ""
  }

  certificate_authentication {
    client_certificate    = octopusdeploy_certificate.certificate_kind_ca.id
  }
}