package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"golang.org/x/sync/errgroup"
)

type BaseWorkerConverter struct {
	Client                   client.OctopusClient
	Excluder                 ExcludeByName
	MachinePolicyConverter   ConverterWithStatelessById
	ExcludeAllWorkers        bool
	ExcludeWorkers           args.StringSliceArgs
	ExcludeWorkersRegex      args.StringSliceArgs
	ExcludeWorkersExcept     args.StringSliceArgs
	ErrGroup                 *errgroup.Group
	LimitResourceCount       int
	IncludeSpaceInPopulation bool
	IncludeIds               bool
	GenerateImportScripts    bool
}
