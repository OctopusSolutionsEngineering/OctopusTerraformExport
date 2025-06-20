resource "octopusdeploy_tenant" "tenant_australian_office" {
  name        = "Australian Office"
  description = "An example tenant that represents an Australian office"
  tenant_tags = ["Cities/Sydney"]
  depends_on  = [octopusdeploy_tag_set.tagset_cities,octopusdeploy_tag.tagset_cities_tag_sydney]
}
