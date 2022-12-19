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
	url, space, apiKey := parseUrl()

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
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = writeFiles(hcl)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func parseUrl() (string, string, string) {
	var url string
	flag.StringVar(&url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")

	var space string
	flag.StringVar(&space, "space", "", "The Octopus space name or ID")

	var apiKey string
	flag.StringVar(&apiKey, "apiKey", "", "The Octopus api key")

	flag.Parse()

	return url, space, apiKey
}

func writeFiles(files map[string]string) error {
	writer := writers.FileWriter{}
	output, err := writer.Write(files)
	if err != nil {
		return err
	}
	fmt.Println(output)
	return nil
}
