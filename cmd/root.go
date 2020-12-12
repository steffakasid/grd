package cmd

/*
Copyright © 2020 Steffen Rumpf <github@steffen-rumpf.de>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile, newName string
	deleteOldDefault bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grd {gitlab_group_ID} [--flags]",
	Short: "grd renames the default branch of all projects in a group",
	Long: `Grd will rename the default branch of all projects within a group.
	Therefore all projects are looked up. If the default branch already matches,
	the new-name (defaults to 'main') it does nothing. Otherwise it will create a
	new branch based on the old default branch. Sets the new branch to default and
	protects it. Afterwards the old default is unprotected. Optionally you can also
	delete the old default.
	
	Examples:
	grd 1234    			- Rename all default branches to 'main' of group 1234 and keep the old default
	grd 1234 --new-name 1st - Rename all default branches to '1st' of group 1234 and keep the old default
	grd 1234 --delete		- Rename all default branches to 'main' of group 1234 and delete the old default`,
	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.grd.yaml)")
	rootCmd.Flags().StringVarP(&newName, "new-name", "n", "main", "Set the newname to rename the default to")
	rootCmd.Flags().BoolVarP(&deleteOldDefault, "delete", "d", false, "Delete the old default branch when done")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".grd" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".grd")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
