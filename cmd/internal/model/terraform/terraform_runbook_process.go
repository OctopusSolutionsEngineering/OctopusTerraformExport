package terraform

type TerraformRunbookProcess struct {
	Type      string          `hcl:"type,label"`
	Name      string          `hcl:"name,label"`
	RunbookId string          `hcl:"runbook_id"`
	Step      []TerraformStep `hcl:"step,block"`
}
