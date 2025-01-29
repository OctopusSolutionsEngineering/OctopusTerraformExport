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

resource "octopusdeploy_ssh_connection_deployment_target" "ssh" {
  account_id            = octopusdeploy_ssh_key_account.account_ec2_sydney.id
  environments          = [octopusdeploy_environment.development_environment.id]
  fingerprint           = "d5:6b:a3:78:fa:fe:f5:ad:d4:79:4a:57:35:6a:32:ef"
  host                  = "3.25.215.87"
  name                  = "Ssh"
  roles                 = ["vm"]
  dot_net_core_platform = "linux-x64"
  machine_policy_id     = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
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
  certificate                       = file("dummycert.txt")
}

# resource "octopusdeploy_azure_cloud_service_deployment_target" "azure" {
#   environments                      = ["${octopusdeploy_environment.development_environment.id}"]
#   name                              = "Azure"
#   roles                             = ["cloud"]
#   account_id                        = "${octopusdeploy_azure_subscription_account.account_subscription.id}"
#   cloud_service_name                = "servicename"
#   storage_account_name              = "accountname"
#   default_worker_pool_id            = ""
#   health_status                     = "Unhealthy"
#   is_disabled                       = false
#   machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
#   shell_name                        = "Unknown"
#   shell_version                     = "Unknown"
#   tenant_tags                       = []
#   tenanted_deployment_participation = "Untenanted"
#   tenants                           = []
#   thumbprint                        = ""
#   use_current_instance_count        = true
# }

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