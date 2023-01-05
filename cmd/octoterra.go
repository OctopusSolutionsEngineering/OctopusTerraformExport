package main

import (
	"flag"
	"fmt"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/converters"
	"github.com/mcasperson/OctopusTerraformExport/internal/writers"
	"os"
)

func main() {
	url, space, apiKey, dest := parseUrl()
	err := ConvertToTerraform(url, space, apiKey, dest)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func ConvertToTerraform(url string, space string, apiKey string, dest string) error {
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

	err = writeFiles(hcl, dest)

	return err
}

func parseUrl() (string, string, string, string) {
	var url string
	flag.StringVar(&url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")

	var space string
	flag.StringVar(&space, "space", "", "The Octopus space name or ID")

	var apiKey string
	flag.StringVar(&apiKey, "apiKey", "", "The Octopus api key")

	var dest string
	flag.StringVar(&dest, "dest", "", "The directory to place the Terraform files in")

	flag.Parse()

	return url, space, apiKey, dest
}

func writeFiles(files map[string]string, dest string) error {
	writer := writers.NewFileWriter(dest)
	//writer := writers.ConsoleWriter{}
	output, err := writer.Write(files)
	if err != nil {
		return err
	}
	fmt.Println(output)
	return nil
}
