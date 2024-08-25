resource "octopusdeploy_ssh_key_account" "account_ssh" {
  description                       = "A test account"
  name                              = "SSH"
  environments                      = null
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  private_key_file                  = "unused"
  username                          = "admin"
  # Because of https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/issues/343
  private_key_passphrase            = file("dummycert.txt")
}