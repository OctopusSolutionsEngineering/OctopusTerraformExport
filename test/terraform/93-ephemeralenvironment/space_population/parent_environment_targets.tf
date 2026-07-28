/*
  An example of each deployment target type, all scoped to a parent environment rather than a
  regular environment. These verify that every target converter resolves parent environments.
*/

data "octopusdeploy_machine_policies" "default_machine_policy" {
  ids          = null
  partial_name = "Default Machine Policy"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_cloud_region_deployment_target" "target_parent_env_cloud_region" {
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Cloud Region"
  roles                             = ["cloud"]
  default_worker_pool_id            = ""
  health_status                     = "Healthy"
  is_disabled                       = false
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
}

resource "octopusdeploy_listening_tentacle_deployment_target" "target_parent_env_listening" {
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Listening"
  roles                             = ["vm"]
  tentacle_url                      = "https://tentacle/"
  thumbprint                        = "55E05FD1B0F76E60F6DA103988056CE695685FD1"
  is_disabled                       = false
  is_in_process                     = false
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []

  tentacle_version_details {
  }
}

resource "octopusdeploy_polling_tentacle_deployment_target" "target_parent_env_polling" {
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Polling"
  roles                             = ["vm"]
  tentacle_url                      = "poll://abcdefghijklmnopqrst/"
  thumbprint                        = "1854A302E5D9EAC1CAA3DA1F5249F82C28BB2B86"
  is_disabled                       = false
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  shell_name                        = "PowerShell"
  shell_version                     = "5.1.22621.1"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []

  tentacle_version_details {
  }
}

resource "octopusdeploy_ssh_connection_deployment_target" "target_parent_env_ssh" {
  account_id            = octopusdeploy_username_password_account.account_parent_env.id
  environments          = [octopusdeploy_parent_environment.example.id]
  fingerprint           = "d5:6b:a3:78:fa:fe:f5:ad:d4:79:4a:57:35:6a:32:ef"
  host                  = "3.25.215.87"
  name                  = "Parent Env SSH"
  roles                 = ["vm"]
  dot_net_core_platform = "linux-x64"
  machine_policy_id     = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
}

resource "octopusdeploy_offline_package_drop_deployment_target" "target_parent_env_offline_drop" {
  applications_directory            = "c:\\temp"
  working_directory                 = "c:\\temp"
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Offline Drop"
  roles                             = ["offline"]
  health_status                     = "Healthy"
  is_disabled                       = false
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
}

resource "octopusdeploy_kubernetes_cluster_deployment_target" "target_parent_env_kubernetes" {
  cluster_url                       = "https://cluster"
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Kubernetes"
  roles                             = ["k8s"]
  cluster_certificate               = ""
  cluster_certificate_path          = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
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

  endpoint {
    communication_style = "Kubernetes"
  }

  pod_authentication {
    token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"
  }
}

resource "octopusdeploy_azure_service_principal" "account_parent_env_service_principal" {
  name                              = "Parent Env Service Principal"
  description                       = "A service principal used by the Azure targets"
  environments                      = [octopusdeploy_parent_environment.example.id]
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  application_id                    = "08a4a027-6f2a-4793-a0e5-e59a3c79189f"
  password                          = "secretgoeshere"
  subscription_id                   = "3b50dcf4-f74d-442e-93cb-301b13e1e2d5"
  tenant_id                         = "3d13e379-e666-469e-ac38-ec6fd61c1166"
}

resource "octopusdeploy_azure_subscription_account" "account_parent_env_subscription" {
  description                       = "A subscription account used by the Azure cloud service target"
  name                              = "Parent Env Subscription"
  environments                      = [octopusdeploy_parent_environment.example.id]
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  storage_endpoint_suffix           = "storage_endpoint_suffix"
  subscription_id                   = "fde6a0ae-a1d4-40ae-91de-88f4ed898c03"
  azure_environment                 = "AzureCloud"
  management_endpoint               = "management_endpoint"
  certificate                       = file("azuresubscriptioncert.txt")
}

resource "octopusdeploy_azure_web_app_deployment_target" "target_parent_env_web_app" {
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Azure Web App"
  roles                             = ["cloud"]
  account_id                        = octopusdeploy_azure_service_principal.account_parent_env_service_principal.id
  resource_group_name               = "mattc-webapp"
  web_app_name                      = "mattc-webapp"
  health_status                     = "Unhealthy"
  is_disabled                       = false
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
  web_app_slot_name                 = "slot1"
}

/*
  There is no Azure cloud service target here. Octopus rejects the creation of new ones with
  "Microsoft has retired Azure Cloud Services, as such no new Azure Cloud Service Deployment
  Targets can be created", so the parent environment support in the converter for that target
  type can not be tested against a current Octopus server.
*/

resource "octopusdeploy_azure_service_fabric_cluster_deployment_target" "target_parent_env_service_fabric" {
  environments                      = [octopusdeploy_parent_environment.example.id]
  name                              = "Parent Env Service Fabric"
  roles                             = ["cloud"]
  connection_endpoint               = "http://endpoint"
  aad_client_credential_secret      = ""
  aad_credential_type               = "UserCredential"
  aad_user_credential_password      = "secretgoeshere"
  aad_user_credential_username      = "username"
  certificate_store_location        = ""
  certificate_store_name            = ""
  client_certificate_variable       = ""
  health_status                     = "Unhealthy"
  is_disabled                       = false
  machine_policy_id                 = data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
  thumbprint                        = ""
}
