package octopus

type TagSet struct {
	Id          string
	Name        string
	Description *string
	SortOrder   int
	Tags        []Tag
}

type Tag struct {
	Id          string
	Name        string
	Color       string
	Description *string
	SortOrder   int
}
