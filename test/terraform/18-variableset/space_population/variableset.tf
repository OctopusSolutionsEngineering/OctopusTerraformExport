resource "octopusdeploy_library_variable_set" "octopus_library_variable_set" {
  name = "Test"
  description = "Test variable set"
}

resource "octopusdeploy_variable" "octopus_admin_api_key" {
  name = "Test.Variable"
  type = "String"
  description = "Test variable"
  is_sensitive = false
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  value = "True"

  prompt {
    description = "test description"
    label       = "test label"
    is_required = true
    display_settings {
      control_type = "Select"
      select_option {
        display_name = "hi"
        value = "there"
      }
    }
  }
}

resource "octopusdeploy_variable" "secret" {
  name = "Test.SecretVariable"
  type = "Sensitive"
  description = "Test variable"
  is_sensitive = true
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  value = "True"
}

resource "octopusdeploy_variable" "tag_scoped" {
  depends_on = [octopusdeploy_tag.tag_a]

  name         = "Test.TagScopedVariable"
  type         = "String"
  description  = "Test variable"
  is_sensitive = false
  is_editable  = true
  owner_id     = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  value        = "True"

  scope {
    tenant_tags = ["tag1/a"]
  }
}

resource "octopusdeploy_library_variable_set" "octopus_library_variable_set2" {
  name = "Test2"
  description = "Test variable set"
}

resource "octopusdeploy_library_variable_set" "octopus_library_variable_set3" {
  name = "Test3"
  description = "Test variable set"
}