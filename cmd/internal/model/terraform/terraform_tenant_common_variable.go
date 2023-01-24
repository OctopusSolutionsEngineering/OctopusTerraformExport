package terraform

type TerraformTenantCommonVariable struct {
	Type                 string  `hcl:"type,label"`
	Name                 string  `hcl:"name,label"`
	Id                   *string `hcl:"id"`
	LibraryVariableSetId string  `hcl:"library_variable_set_id"`
	TemplateId           string  `hcl:"template_id"`
	TenantId             string  `hcl:"tenant_id"`
	Value                *string `hcl:"value"`
}
