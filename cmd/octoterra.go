package main

import (
	"flag"
	"fmt"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/converters"
	"github.com/mcasperson/OctopusTerraformExport/internal/singleconverter"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"github.com/mcasperson/OctopusTerraformExport/internal/writers"
	"os"
)

func main() {
	url, space, apiKey, dest, console, projectId := parseUrl()

	var err error = nil

	if projectId != "" {
		err = ConvertProjectToTerraform(url, space, apiKey, dest, console, projectId)
	} else {
		err = ConvertSpaceToTerraform(url, space, apiKey, dest, console)
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func ConvertSpaceToTerraform(url string, space string, apiKey string, dest string, console bool) error {
	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	spaceConverter := converters.SpaceConverter{
		Client: client,
	}

	hcl, err := spaceConverter.ToHcl()

	if err != nil {
		return err
	}

	err = writeFiles(util.UnEscapeDollar(hcl), dest, console)

	return err
}

func ConvertProjectToTerraform(url string, space string, apiKey string, dest string, console bool, projectId string) error {
	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	converter := singleconverter.SingleProjectConverter{
		Client: client,
	}

	dependencies := singleconverter.ResourceDetailsCollection{}
	err := converter.ToHclById(projectId, &dependencies)

	if err != nil {
		return err
	}

	hcl, err := processResources(dependencies.Resources)

	if err != nil {
		return err
	}

	err = writeFiles(util.UnEscapeDollar(hcl), dest, console)

	return err
}

func processResources(resources []singleconverter.ResourceDetails) (map[string]string, error) {
	resourceMap := map[string]singleconverter.ResourceDetails{}
	fileMap := map[string]string{}

	for _, r := range resources {
		resourceMap[r.ResourceType+r.Id] = r
	}

	for _, r := range resources {
		// Some resources are already resolved by their parent, but exist in the resource details map as a lookup.
		// In these cases, ToHcl is nil.
		if r.ToHcl == nil {
			continue
		}

		hcl, err := r.ToHcl(resourceMap)

		if err != nil {
			return nil, err
		}

		fileMap[r.FileName] = hcl
	}

	return fileMap, nil
}

func parseUrl() (string, string, string, string, bool, string) {
	var url string
	flag.StringVar(&url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")

	var space string
	flag.StringVar(&space, "space", "", "The Octopus space name or ID")

	var apiKey string
	flag.StringVar(&apiKey, "apiKey", "", "The Octopus api key")

	var dest string
	flag.StringVar(&dest, "dest", "", "The directory to place the Terraform files in")

	var console bool
	flag.BoolVar(&console, "console", false, "Dump Terraform files to the console")

	var projectId string
	flag.StringVar(&projectId, "projectId", "", "Limit the export to a single project")

	flag.Parse()

	return url, space, apiKey, dest, console, projectId
}

func writeFiles(files map[string]string, dest string, console bool) error {
	writer := writers.NewFileWriter(dest)
	output, err := writer.Write(files)
	if err != nil {
		return err
	}
	fmt.Println(output)

	if console {
		consoleWriter := writers.ConsoleWriter{}
		output, err = consoleWriter.Write(files)
		if err != nil {
			return err
		}
		fmt.Println(output)
	}

	return nil
}
