package terraform

type TerraformTagSet struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Count        *string `hcl:"count"`
	ResourceName string  `hcl:"name"`
	Description  *string `hcl:"description"`
	SortOrder    int     `hcl:"sort_order"`
}

type TerraformTag struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Count        *string `hcl:"count"`
	ResourceName string  `hcl:"name"`
	TagSetId     string  `hcl:"tag_set_id"`
	Color        string  `hcl:"color"`
	Description  *string `hcl:"description"`
	SortOrder    int     `hcl:"sort_order"`
}
