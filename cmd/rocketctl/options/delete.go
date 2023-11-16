package options

import (
	"fmt"

	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/rocketctl/applications"
	"github.com/hex-techs/rocket/pkg/rocketctl/clusters"
	"github.com/hex-techs/rocket/pkg/rocketctl/modules"
	"github.com/spf13/cobra"
)

var Delete = &cobra.Command{
	Use:              "delete",
	TraverseChildren: true,
	Short:            "delete resource from rocket",
	Long: `Delete resource from rocket, you can use the command to delete resource from rocket.
	
- delete rocket:

  rctl delete
`,
	Run: func(cmd *cobra.Command, args []string) {
		resource, ok := validateArgs(args)
		if !ok {
			return
		}
		if len(args) != 2 {
			if len(args) < 2 {
				fmt.Printf("wrong args, must specify the resource name\n")
				return
			}
			if len(args) > 2 {
				fmt.Printf("Unknown args: %+v\n", args[2:])
				return
			}
		}
		id := args[1]
		deleteResource(resource, id)
	},
}

func deleteResource(resource, id string) {
	var err error
	switch resource {
	case "module":
		err = modules.Delete(Path, id)
		cobra.CheckErr(err)
	case "cluster":
		var cls cluster.Cluster
		err = parseResourceFile(&cls)
		cobra.CheckErr(err)
		err = clusters.Create(Path, &cls)
		cobra.CheckErr(err)
	case "application":
		var app application.Application
		err = parseResourceFile(&app)
		cobra.CheckErr(err)
		err = applications.Create(Path, &app)
		cobra.CheckErr(err)
	}
}
