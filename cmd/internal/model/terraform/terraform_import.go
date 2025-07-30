package terraform

type TerraformImport struct {
	To string `hcl:"to"`
	Id string `hcl:"id"`
}
