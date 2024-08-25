resource "octopusdeploy_ssh_key_account" "account_ssh" {
  description                       = "A test account"
  name                              = "SSH"
  environments                      = null
  tenanted_deployment_participation = "Tenanted"
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type with space/a with space"]
  tenants                           = null
  private_key_file                  = "unused"
  username                          = "admin"
  # Because of https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/issues/343
  private_key_passphrase            = file("dummycert3.txt")
}