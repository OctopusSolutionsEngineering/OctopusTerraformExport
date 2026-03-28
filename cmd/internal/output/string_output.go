package output

import (
	"strings"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
)

func WriteString(files map[string]string) string {
	var sb strings.Builder
	for _, str := range strutil.UnEscapeDollarInMap(files) {
		sb.WriteString(str + "\n\n")
	}
	return sb.String()
}
