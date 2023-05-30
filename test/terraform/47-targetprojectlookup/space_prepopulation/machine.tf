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

resource "octopusdeploy_ssh_key_account" "account_ec2_sydney" {
  name                              = "ec2 sydney"
  description                       = ""
  environments                      = null
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  private_key_file                  = "whatever"
  username                          = "ec2-user"
  private_key_passphrase            = "whatever"
}

resource "octopusdeploy_ssh_connection_deployment_target" "target_3_25_215_87" {
  account_id            = "${octopusdeploy_ssh_key_account.account_ec2_sydney.id}"
  environments          = ["${octopusdeploy_environment.development_environment.id}"]
  fingerprint           = "d5:6b:a3:78:fa:fe:f5:ad:d4:79:4a:57:35:6a:32:ef"
  host                  = "3.25.215.87"
  name                  = "Ssh"
  roles                 = ["vm"]
  dot_net_core_platform = "linux-x64"
  machine_policy_id     = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
}

resource "octopusdeploy_listening_tentacle_deployment_target" "listening" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "Listening"
  roles                             = ["vm"]
  tentacle_url                      = "https://tentacle/"
  thumbprint                        = "55E05FD1B0F76E60F6DA103988056CE695685FD1"
  is_disabled                       = false
  is_in_process                     = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []

  tentacle_version_details {
  }
}

resource "octopusdeploy_polling_tentacle_deployment_target" "polling" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "Polling"
  roles                             = ["vm"]
  tentacle_url                      = "poll://abcdefghijklmnopqrst/"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "PowerShell"
  shell_version                     = "5.1.22621.1"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []

  tentacle_version_details {
  }

  thumbprint = "1854A302E5D9EAC1CAA3DA1F5249F82C28BB2B86"
}

resource "octopusdeploy_offline_package_drop_deployment_target" "target_offlineoffline" {
  applications_directory            = "c:\\temp"
  working_directory                 = "c:\\temp"
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "Offline"
  roles                             = ["offline"]
  health_status                     = "Healthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
}

resource "octopusdeploy_azure_subscription_account" "account_subscription" {
  description                       = "A test account"
  name                              = "Subscription"
  environments                      = null
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  storage_endpoint_suffix           = "whatever"
  subscription_id                   = "fde6a0ae-a1d4-40ae-91de-88f4ed898c03"
  azure_environment                 = "AzureCloud"
  management_endpoint               = "whatever"
  certificate                       = "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"
}

resource "octopusdeploy_azure_cloud_service_deployment_target" "azure" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "Azure"
  roles                             = ["cloud"]
  account_id                        = "${octopusdeploy_azure_subscription_account.account_subscription.id}"
  cloud_service_name                = "servicename"
  storage_account_name              = "accountname"
  default_worker_pool_id            = ""
  health_status                     = "Unhealthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
  use_current_instance_count        = true
}

resource "octopusdeploy_azure_service_fabric_cluster_deployment_target" "target_service_fabric" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "ServiceFabric"
  roles                             = ["cloud"]
  connection_endpoint               = "http://endpoint"
  aad_client_credential_secret      = ""
  aad_credential_type               = "UserCredential"
  aad_user_credential_password      = "passwword"
  aad_user_credential_username      = "username"
  certificate_store_location        = ""
  certificate_store_name            = ""
  client_certificate_variable       = ""
  health_status                     = "Unhealthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
}

resource "octopusdeploy_azure_service_principal" "account_sales_account" {
  name                              = "Sales Account"
  description                       = ""
  environments                      = null
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  application_id                    = "08a4a027-6f2a-4793-a0e5-e59a3c79189f"
  password                          = "Password"
  subscription_id                   = "3b50dcf4-f74d-442e-93cb-301b13e1e2d5"
  tenant_id                         = "3d13e379-e666-469e-ac38-ec6fd61c1166"
}

resource "octopusdeploy_azure_web_app_deployment_target" "target_web_app" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "WebApp"
  roles                             = ["cloud"]
  account_id                        = "${octopusdeploy_azure_service_principal.account_sales_account.id}"
  resource_group_name               = "mattc-webapp"
  web_app_name                      = "mattc-webapp"
  health_status                     = "Unhealthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
  web_app_slot_name                 = "slot1"
}

resource "octopusdeploy_cloud_region_deployment_target" "target_region1" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "Cloud"
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