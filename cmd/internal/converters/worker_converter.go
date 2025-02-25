package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"golang.org/x/sync/errgroup"
)

type WorkerConverter struct {
	Client                           client.OctopusClient
	ExcludeEnvironments              args.StringSliceArgs
	ExcludeEnvironmentsRegex         args.StringSliceArgs
	ExcludeEnvironmentsExcept        args.StringSliceArgs
	ExcludeAllEnvironments           bool
	ExcludeWorkersWithNoEnvironments bool
	Excluder                         ExcludeByName
	MachinePolicyConverter           ConverterWithStatelessById
	AccountConverter                 ConverterAndLookupWithStatelessById
	EnvironmentConverter             ConverterAndLookupWithStatelessById
	CertificateConverter             ConverterAndLookupWithStatelessById
	ExcludeAllWorkers                bool
	ExcludeWorkers                   args.StringSliceArgs
	ExcludeWorkersRegex              args.StringSliceArgs
	ExcludeWorkersExcept             args.StringSliceArgs
	ErrGroup                         *errgroup.Group
	LimitResourceCount               int
	IncludeSpaceInPopulation         bool
	IncludeIds                       bool
	GenerateImportScripts            bool
}
