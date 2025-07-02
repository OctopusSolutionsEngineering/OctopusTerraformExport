resource "octopusdeploy_tenant" "tenant_main_office" {
  name        = "Main Office"
  description = "An example tenant that represents that main US office"
  tenant_tags = ["Cities/Washington"]
  depends_on  = [octopusdeploy_tag.tagset_cities_tag_washington,octopusdeploy_tag_set.tagset_cities]
}
