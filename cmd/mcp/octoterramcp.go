package main

import (
	"context"
	"log"
	"os"
	"reflect"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/output"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Output struct {
	Terraform string `json:"terraform" jsonschema:"the generated Terraform configuration"`
}

// buildInputSchema generates a JSON schema for args.Arguments with all fields
// marked as optional (Required list cleared). StringSliceArgs is mapped to an
// array-of-strings schema so the MCP client can pass it as a JSON array.
func buildInputSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[args.Arguments](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[args.StringSliceArgs](): {
				Type:  "array",
				Items: &jsonschema.Schema{Type: "string"},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if schema != nil {
		schema.Required = nil
	}

	return schema, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "Octoterra", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "convertOctopusToTerraform",
		Description: "Convert Octopus space or project to Terraform configuration",
		InputSchema: buildInputSchema(),
	}, convert)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func convert(ctx context.Context, req *mcp.CallToolRequest, input args.Arguments) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	// These arguments don't make sense or can have default values
	input.ApiKey = os.Getenv("OCTOPUS_CLI_API_KEY")
	input.AccessToken = ""
	input.Url = os.Getenv("OCTOPUS_CLI_SERVER")
	input.UseRedirector = false
	input.Console = true
	input.ConfigFile = ""
	input.ConfigPath = ""
	input.Version = false
	input.Profiling = false
	input.ExcludeSpaceCreation = true

	// Ignore things that look like empty arrays
	if input.RunbookName == "[]" {
		input.RunbookName = ""
	}

	if input.RunbookId == "[]" {
		input.RunbookId = ""
	}

	if input.Destination == "" {
		dir, err := os.MkdirTemp("", "octoterra*")
		if err != nil {
			return nil, Output{}, err
		}
		input.Destination = dir
	}

	files, err := entry.Entry(input, "")

	if err != nil {
		return nil, Output{}, err
	}

	result := output.WriteString(files)

	return nil, Output{Terraform: result}, nil
}
