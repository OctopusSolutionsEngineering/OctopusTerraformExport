data "octopusdeploy_machine_policies" "default_machine_policy" {
  ids          = null
  partial_name = "Default Machine Policy"
  skip         = 0
  take         = 1
}

resource octopusdeploy_kubernetes_cluster_deployment_target test_eks{
  cluster_url                       = "https://cluster"
  environments                      = ["${octopusdeploy_environment.test_environment.id}"]
  name                              = "Test"
  roles                             = ["eks"]
  cluster_certificate               = ""
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  namespace                         = ""
  skip_tls_verification             = true
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
  uri                               = ""

  endpoint {
    communication_style    = "Kubernetes"
    cluster_certificate    = ""
    cluster_url            = "https://cluster"
    namespace              = ""
    skip_tls_verification  = true
    default_worker_pool_id = ""
  }

  container {
    feed_id = ""
    image   = ""
  }

  aws_account_authentication {
    account_id        = "${octopusdeploy_aws_account.account_aws_account.id}"
    cluster_name      = "clustername"
    assume_role       = false
    use_instance_role = false
  }
}

resource "octopusdeploy_cloud_region_deployment_target" "target_region1" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "CloudRegion"
  roles                             = ["cloud"]
  default_worker_pool_id            = ""
  health_status                     = "Healthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
}