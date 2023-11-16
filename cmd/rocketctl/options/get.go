package options

import (
	"fmt"
	"os"

	"github.com/hex-techs/rocket/pkg/rocketctl/applications"
	"github.com/hex-techs/rocket/pkg/rocketctl/clusters"
	"github.com/hex-techs/rocket/pkg/rocketctl/modules"
	"github.com/hex-techs/rocket/pkg/rocketctl/projects"
	"github.com/spf13/cobra"
)

var output string

var Get = &cobra.Command{
	Use:              "get",
	TraverseChildren: true,
	Short:            "get resource from rocket",
	Long: `Get resource from rocket, you can use the command to get resource from rocket.
	
- get rocket:

  rctl get
`,
	Run: func(cmd *cobra.Command, args []string) {
		resource, ok := validateArgs(args)
		if !ok {
			os.Exit(1)
		}
		if len(args) == 1 {
			switch resource {
			case "module":
				err := modules.List(Path)
				cobra.CheckErr(err)
			case "project":
				err := projects.List(Path)
				cobra.CheckErr(err)
			case "cluster":
				err := clusters.List(Path)
				cobra.CheckErr(err)
			case "application":
				err := applications.List(Path)
				cobra.CheckErr(err)
			}
			return
		}
		name := args[1]
		switch resource {
		case "user":
			fmt.Printf("get user: %s\n", name)
		case "config":
			fmt.Printf("get config: %s\n", name)
		case "cluster":
			err := clusters.Get(Path, name)
			cobra.CheckErr(err)
		case "application":
			err := applications.Get(Path, name)
			cobra.CheckErr(err)
		}
	},
}

func init() {
	Get.Flags().StringVarP(&output, "output", "o", "json", "output format, support json, yaml")
}
