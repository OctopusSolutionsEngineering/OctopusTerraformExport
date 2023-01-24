package terraform

type TerraformFeedData struct {
	Type        string   `hcl:"type,label"`
	Name        string   `hcl:"name,label"`
	FeedType    string   `hcl:"feed_type"`
	Ids         []string `hcl:"ids"`
	PartialName string   `hcl:"partial_name"`
	Skip        int      `hcl:"skip"`
	Take        int      `hcl:"take"`
}
