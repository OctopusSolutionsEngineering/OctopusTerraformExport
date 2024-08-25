resource "octopusdeploy_certificate" "certificate_kind_ca" {
  name                              = "Test"
  certificate_data                  = file("dummycert.txt")
  password                          = "Password01!"
  environments                      = []
  notes                             = "A test certificate"
  tenant_tags                       = ["type with space/a with space"]
  tenanted_deployment_participation = "Tenanted"
  tenants                           = []
  depends_on                        = [octopusdeploy_tag.tag_a]
}