package args

type Arguments struct {
	Url                       string
	ApiKey                    string
	Space                     string
	Destination               string
	Console                   bool
	ProjectId                 string
	ProjectName               string
	LookupProjectDependencies bool
	IgnoreCacManagedValues    bool
	BackendBlock              string
	FlattenProjectTemplates   bool
}
