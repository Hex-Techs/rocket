package options

import (
	"fmt"

	"github.com/hex-techs/rocket/pkg/rocketctl/cerificateauthority"
	"github.com/spf13/cobra"
)

var (
	addr     string
	username string
	password string
)

var Login = &cobra.Command{
	Use:              "login",
	TraverseChildren: true,
	Short:            "sign in to rocket",
	Long: `Sign in to rocket, you can use the command to sign in to rocket.
	
- sing in to rocket:

  rctl login -u <username> -p <password>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cerificateauthority.Login(Path, addr, username, password); err != nil {
			cobra.CheckErr(err)
			return
		}
		fmt.Println("login successfully")
	},
}

func init() {
	Login.Flags().StringVarP(&addr, "address", "a", "http://localhost:9100", "Specify the address of the rocket manager, default: (http://localhost:9100)")
	Login.Flags().StringVarP(&username, "username", "u", "", "Specify the username (required)")
	Login.Flags().StringVarP(&password, "password", "p", "", "Specify the password (required)")

	Login.MarkFlagRequired("username")
	Login.MarkFlagRequired("password")
}
