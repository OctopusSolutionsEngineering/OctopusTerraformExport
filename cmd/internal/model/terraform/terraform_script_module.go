package terraform

type TerraformScriptModule struct {
	Type         string                      `hcl:"type,label"`
	Name         string                      `hcl:"name,label"`
	Count        *string                     `hcl:"count"`
	Description  *string                     `hcl:"description"`
	ResourceName string                      `hcl:"name"`
	Script       TerraformScriptModuleScript `hcl:"script,block"`
}

type TerraformScriptModuleScript struct {
	Body   string `hcl:"body"`
	Syntax string `hcl:"syntax"`
}
