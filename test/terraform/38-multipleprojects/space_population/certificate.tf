resource "octopusdeploy_certificate" "certificate_kind_ca" {
  name                              = "Test"
  certificate_data                  = file("dummycert.txt")
  password                          = "Password01!"
  environments                      = []
  notes                             = "A test certificate"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
}