package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hex-techs/rocket/cmd/rocketctl/options"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rctl",
	Short: "rocket command line",
	Long:  `rocket is a CLI for rocket system.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&options.Path, "conf", "", filepath.Join(homeDir(), ".config/rocket"),
		"Specify the path to save the configuration file, default: (~/.config/rocket)")
	rootCmd.PersistentFlags().StringVarP(&options.ResourceFile, "file", "f", "", "Specify the resource file")
	rootCmd.PersistentFlags().StringVarP(&options.Cluster, "cluster", "c", "", "Specify the cluster name")
	rootCmd.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "default", "Specify the namespace")

	rootCmd.AddCommand(options.Version, options.Login, options.Approve, options.Get, options.Create, options.Delete)
	cobra.OnInitialize(initConfig)
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	Execute()
}
