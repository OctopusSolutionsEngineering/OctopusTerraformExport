package terraform

type TerraformOutput struct {
	Name  string `hcl:"name,label"`
	Value string `hcl:"value"`
}
