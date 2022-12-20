package model

type TerraformVariable struct {
	Name        string `hcl:"name,label"`
	Type        string `hcl:"type"`
	Nullable    bool   `hcl:"nullable"`
	Sensitive   bool   `hcl:"sensitive"`
	Description string `hcl:"description"`
}
