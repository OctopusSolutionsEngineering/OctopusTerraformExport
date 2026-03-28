package main

import (
	"context"
	"log"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/output"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Output struct {
	Terraform string `json:"terraform" jsonschema:"the generated Terraform configuration"`
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "Octoterra", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "convertOctopusToTerraform", Description: "Convert Octopus space or project to Terraform configuration"}, convert)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func convert(ctx context.Context, req *mcp.CallToolRequest, input args.Arguments) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	files, err := entry.Entry(input, "")

	if err != nil {
		return nil, Output{}, err
	}

	result := output.WriteString(files)

	return nil, Output{Terraform: result}, nil
}
