package options

import (
	"fmt"

	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/models/module"
	"github.com/hex-techs/rocket/pkg/models/project"
	"github.com/hex-techs/rocket/pkg/rocketctl/applications"
	"github.com/hex-techs/rocket/pkg/rocketctl/clusters"
	"github.com/hex-techs/rocket/pkg/rocketctl/modules"
	"github.com/hex-techs/rocket/pkg/rocketctl/projects"
	"github.com/spf13/cobra"
)

var Create = &cobra.Command{
	Use:              "create",
	TraverseChildren: true,
	Short:            "create resource to rocket",
	Long: `Create resource to rocket, you can use the command to create resource to rocket.
	
- create rocket:

  rctl create
`,
	Run: func(cmd *cobra.Command, args []string) {
		resource, ok := validateArgs(args)
		if !ok {
			return
		}
		if len(args) > 1 {
			fmt.Printf("Unknown args: %+v\n", args[1:])
			return
		}
		createResource(resource)
	},
}

func createResource(resource string) {
	var err error
	switch resource {
	case "module":
		var m module.Module
		err = parseResourceFile(&m)
		cobra.CheckErr(err)
		err = modules.Create(Path, &m)
		cobra.CheckErr(err)
	case "project":
		var p project.Project
		err = parseResourceFile(&p)
		cobra.CheckErr(err)
		err = projects.Create(Path, &p)
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
