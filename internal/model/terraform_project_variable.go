package model

import "github.com/hashicorp/hcl2/hcl"

type TerraformProjectVariable struct {
	Type           string                         `hcl:"type,label"`
	Name           string                         `hcl:"name,label"`
	OwnerId        string                         `hcl:"owner_id"`
	Value          *string                        `hcl:"value"`
	ResourceName   *string                        `hcl:"name"`
	Description    *string                        `hcl:"description"`
	SensitiveValue hcl.Expression                 `hcl:"sensitive_value"`
	IsSensitive    bool                           `hcl:"is_sensitive"`
	Prompt         TerraformProjectVariablePrompt `hcl:"prompt,block"`
}

type TerraformProjectVariablePrompt struct {
	Description *string `hcl:"description"`
	Label       *string `hcl:"label"`
	IsRequired  bool    `hcl:"is_required"`
}
