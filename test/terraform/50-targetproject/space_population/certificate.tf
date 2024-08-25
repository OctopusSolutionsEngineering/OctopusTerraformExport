resource "octopusdeploy_certificate" "certificate_kind_ca" {
  name                              = "Test"
  certificate_data                  = file("dummycert.txt")
  password                          = "Password01!"
  environments                      = [octopusdeploy_environment.test_environment.id]
  notes                             = "A test certificate"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
}