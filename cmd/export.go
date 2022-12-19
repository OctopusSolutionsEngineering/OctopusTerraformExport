package main

import (
	"flag"
	"fmt"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"os"
)

func main() {
	url, space, apiKey := parseUrl()

	client := internal.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	spaceConverter := internal.SpaceConverter{
		Client: client,
	}

	hcl, err := spaceConverter.ToHcl()

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(hcl)
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
