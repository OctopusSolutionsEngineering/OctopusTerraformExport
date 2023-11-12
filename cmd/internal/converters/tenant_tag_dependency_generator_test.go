package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"testing"
)

func TestTagSetDependencies(t *testing.T) {
	tenantTagDependencyGenerator := TenantTagDependencyGenerator{}
	tags := octopus.GeneralCollection[octopus.TagSet]{}
	tags.Items = []octopus.TagSet{
		{
			Id:          "TagSet-1",
			Name:        "tags",
			Description: nil,
			SortOrder:   0,
			Tags: []octopus.Tag{
				{
					Id:               "Tag-1",
					Name:             "a",
					CanonicalTagName: "tags/a",
					Color:            "blue",
					Description:      nil,
					SortOrder:        0,
				},
				{
					Id:               "Tag-2",
					Name:             "b",
					CanonicalTagName: "tags/b",
					Color:            "blue",
					Description:      nil,
					SortOrder:        0,
				},
			},
		},
		{
			Id:          "TagSet-2",
			Name:        "tags2",
			Description: nil,
			SortOrder:   0,
			Tags: []octopus.Tag{
				{
					Id:               "Tag-3",
					Name:             "a",
					CanonicalTagName: "tags2/a",
					Color:            "blue",
					Description:      nil,
					SortOrder:        0,
				},
				{
					Id:               "Tag-4",
					Name:             "b",
					CanonicalTagName: "tags2/b",
					Color:            "blue",
					Description:      nil,
					SortOrder:        0,
				},
			},
		},
	}

	tagSetsDependencies, tagDependencies, err := tenantTagDependencyGenerator.FindDependencies([]string{"tags/a"}, tags)

	if err != nil {
		t.Fatalf("Failed to find dependencies")
	}

	if len(tagSetsDependencies) != 1 {
		t.Fatalf("Failed to find tagsets")
	}

	if tagSetsDependencies[0].Id != "TagSet-1" {
		t.Fatalf("Failed to find correct tagset")
	}

	if len(tagDependencies) != 1 {
		t.Fatalf("Failed to find tags")
	}

	if tagDependencies[0].Id != "Tag-1" {
		t.Fatalf("Failed to find correct tagset")
	}
}
