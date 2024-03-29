package terraform

type TerraformAccountData struct {
	Type        string                          `hcl:"type,label"`
	Name        string                          `hcl:"name,label"`
	Ids         []string                        `hcl:"ids"`
	PartialName string                          `hcl:"partial_name"`
	Skip        int                             `hcl:"skip"`
	Take        int                             `hcl:"take"`
	AccountType *string                         `hcl:"account_type"`
	Lifecycle   *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}
