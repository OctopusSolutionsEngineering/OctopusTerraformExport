package terraform

type TerraformCommunityStepTemplate struct {
	Type                      string  `hcl:"type,label"`
	Name                      string  `hcl:"name,label"`
	CommunityActionTemplateId string  `hcl:"community_action_template_id"`
	Count                     *string `hcl:"count"`
}
