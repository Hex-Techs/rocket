package options

import (
	"fmt"
	"os"

	"github.com/hex-techs/rocket/pkg/rocketctl/clusters"
	"github.com/spf13/cobra"
)

var Approve = &cobra.Command{
	Use:              "approved",
	TraverseChildren: true,
	Short:            "get resource from rocket",
	Long: `Get resource from rocket, you can use the command to get resource from rocket.
	
- get rocket:

  rctl get
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Please specify the cluster name")
			os.Exit(1)
		}
		name := args[0]
		if err := clusters.Approved(Path, name); err != nil {
			cobra.CheckErr(err)
		}
		fmt.Printf("approved cluster: %s\n", name)
	},
}
