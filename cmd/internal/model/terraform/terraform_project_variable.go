package terraform

type TerraformProjectVariable struct {
	Type           string                          `hcl:"type,label"`
	Name           string                          `hcl:"name,label"`
	OwnerId        string                          `hcl:"owner_id"`
	Value          *string                         `hcl:"value"`
	ResourceName   string                          `hcl:"name"`
	ResourceType   string                          `hcl:"type"`
	Description    *string                         `hcl:"description"`
	SensitiveValue *string                         `hcl:"sensitive_value"`
	IsSensitive    bool                            `hcl:"is_sensitive"`
	Prompt         *TerraformProjectVariablePrompt `hcl:"prompt,block"`
	Scope          *TerraformProjectVariableScope  `hcl:"scope,block"`
}

type TerraformProjectVariablePrompt struct {
	Description *string `hcl:"description"`
	Label       *string `hcl:"label"`
	IsRequired  bool    `hcl:"is_required"`
}

type TerraformProjectVariableScope struct {
	Actions      []string `hcl:"actions"`
	Channels     []string `hcl:"channels"`
	Environments []string `hcl:"environments"`
	Machines     []string `hcl:"machines"`
	Roles        []string `hcl:"roles"`
	TenantTags   []string `hcl:"tenant_tags"`
}
