package main

import (
	"context"
	"log"
	"os"
	"reflect"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/output"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
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
	schema, err := buildInputSchema()

	if err != nil {
		log.Fatal(err)
		return
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "convertOctopusToTerraform",
		Description: "Convert Octopus space or project to Terraform configuration",
		InputSchema: schema,
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
	input.ApiKey = strutil.StrPointer(os.Getenv("OCTOPUS_CLI_API_KEY"))
	input.AccessToken = nil
	input.Url = strutil.StrPointer(os.Getenv("OCTOPUS_CLI_SERVER"))
	input.UseRedirector = boolutil.BoolPtr(false)
	input.Console = boolutil.BoolPtr(true)
	input.ConfigFile = nil
	input.ConfigPath = nil
	input.Version = boolutil.BoolPtr(false)
	input.Profiling = boolutil.BoolPtr(false)
	input.ExcludeSpaceCreation = boolutil.BoolPtr(true)
	input.InsecureTls = boolutil.BoolPtr(true)

	// Ignore things that look like empty arrays
	if *input.RunbookName == "[]" {
		input.RunbookName = nil
	}

	if *input.RunbookId == "[]" {
		input.RunbookId = nil
	}

	if *input.Destination == "" {
		dir, err := os.MkdirTemp("", "octoterra*")
		if err != nil {
			return nil, Output{}, err
		}
		input.Destination = strutil.StrPointer(dir)
	}

	files, err := entry.Entry(input, "")

	if err != nil {
		return nil, Output{}, err
	}

	result := output.WriteString(files)

	return nil, Output{Terraform: result}, nil
}
