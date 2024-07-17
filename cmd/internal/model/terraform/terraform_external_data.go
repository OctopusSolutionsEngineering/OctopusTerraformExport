package terraform

// https://registry.terraform.io/providers/hashicorp/external/latest/docs/data-sources/external
type TerraformExternalData struct {
	Type    string            `hcl:"type,label"`
	Name    string            `hcl:"name,label"`
	Program []string          `hcl:"program"`
	Query   map[string]string `hcl:"query"`
}
