package terraform

type TerraformArtifactoryFeed struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Count        *string `hcl:"count"`
	Id           *string `hcl:"id"`
	FeedUri      string  `hcl:"feed_uri"`
	ResourceName string  `hcl:"name"`
	Password     *string `hcl:"password"`
	Username     *string `hcl:"username"`
	Repository   string  `hcl:"repository"`
	LayoutRegex  *string `hcl:"layout_regex"`
}
