package main

import (
	"errors"
	"flag"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/logger"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/output"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"go.uber.org/zap"
	"os"
)

var Version = "development"

func main() {
	logger.BuildLogger()

	parseArgs, argsErrors, err := args.ParseArgs(os.Args[1:])

	if errors.Is(err, flag.ErrHelp) {
		zap.L().Error(argsErrors)
		os.Exit(2)
	} else if err != nil {
		zap.L().Error("got error: " + err.Error())
		zap.L().Error("argsErrors:\n" + argsErrors)
		os.Exit(1)
	}

	if parseArgs.Version {
		zap.L().Info("Version: " + Version)
		os.Exit(0)
	}

	if parseArgs.Url == "" {
		errorExit("You must specify the URL with the -url argument")
	}

	if parseArgs.ApiKey == "" {
		errorExit("You must specify the API key with the -apiKey argument")
	}

	if parseArgs.RunbookName != "" && len(parseArgs.ProjectName) != 1 && len(parseArgs.ProjectId) == 1 {
		errorExit("runbookName requires either a single projectId or projectName to be set")
	}

	if parseArgs.Stateless {
		if parseArgs.StepTemplateKey == "" {
			errorExit("stepTemplate requires stepTemplateKey to be defined (e.g. EKS, AKS, Lambda, WebApp)")
		}

		if parseArgs.StepTemplateName == "" {
			errorExit("stepTemplate requires stepTemplateName to be defined")
		}
	}

	if !parseArgs.ExcludeCaCProjectSettings && parseArgs.ExcludeAllGitCredentials {
		errorExit("excludeAllGitCredentials requires excludeCaCProjectSettings to be true")
	}

	if parseArgs.LookupProjectDependencies && parseArgs.Stateless {
		errorExit("lookupProjectDependencies can not be used with stepTemplate")
	}

	files, err := entry.Entry(parseArgs)

	if err != nil {
		errorExit(err.Error())
	}

	err = output.WriteFiles(strutil.UnEscapeDollarInMap(files), parseArgs.Destination, parseArgs.Console)

	if err != nil {
		errorExit(err.Error())
	}
}

func errorExit(message string) {
	if len(message) == 0 {
		message = "No error message provided"
	}
	zap.L().Error(message)
	os.Exit(1)
}
