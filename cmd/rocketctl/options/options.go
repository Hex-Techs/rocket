package options

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	Path         string
	ResourceFile string
	Cluster      string
	Namespace    string
)

const version = `
                __  .__   
_______   _____/  |_|  |  
\_  __ \_/ ___\   __\  |  
 |  | \/\  \___|  | |  |__
 |__|    \___  >__| |____/
             \/           
`

var Version = &cobra.Command{
	Use:              "version",
	TraverseChildren: true,
	Short:            "show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("rctl is a CLI for rocket system.\n\nVersion: %s\n%s", "v1.0.0", version)
	},
}

var (
	resourceMap = map[string]struct{}{
		"user":        {},
		"config":      {},
		"cluster":     {},
		"application": {},
		"module":      {},
		"project":     {},
		"resource":    {},
		"distributor": {},
		"revision":    {},
	}
)

func validateArgs(args []string) (string, bool) {
	if len(args) == 0 {
		fmt.Println("Please specify the resource")
		return "", false
	}
	resource := args[0]
	if _, ok := resourceMap[resource]; !ok {
		fmt.Printf("Unknown resource: %s\n", resource)
		return "", false
	}
	return resource, true
}

// base on the resource file path to parse the resource file
func parseResourceFile(r interface{}) error {
	f, err := os.OpenFile(ResourceFile, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, r); err != nil {
		return err
	}
	return nil
}
