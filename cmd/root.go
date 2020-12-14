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
	"os"

	"github.com/spf13/cobra"

	"github.com/xanzy/go-gitlab"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	gitLabAdapter *gitlab.Client
	versionFlag   bool
)

var version = "on-dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grd {gitlab_group_ID} [--flags]",
	Args:  cobra.MaximumNArgs(1),
	Short: "grd renames the default branch of all projects in a group",
	Long: `Grd will rename the default branch of all projects within a group.
Therefore all projects are looked up. If the default branch already matches,
the new-name (defaults to 'main') it does nothing. Otherwise it will create a
new branch based on the old default branch. Sets the new branch to default and
protects it. Afterwards the old default is unprotected. Optionally you can also
delete the old default.

Each flag could also be set as env var prefixed with GRD_ e.g. to set the token you 
can do 'export GRD_TOKEN=<your token>'.

Examples:
grd 1234    			      - Rename all default branches to 'main' of group 1234 and keep the old default
grd 1234 --new-name 1st - Rename all default branches to '1st' of group 1234 and keep the old default
grd 1234 --delete		    - Rename all default branches to 'main' of group 1234 and delete the old default
grd 1234 --unprotect    - Rename all default branches to 'main' of group 1234 and unprotect the old default`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if versionFlag {
			fmt.Println("grd version:", version)
			os.Exit(0)
		}

		if len(args) != 1 {
			fmt.Println("You must provide a GitLab GroupID!")
			os.Exit(1)
		}

		gitLabAdapter, err = gitlab.NewClient(viper.GetString("token"), gitlab.WithBaseURL(viper.GetString("url")))
		doWePanic(err)

		group, _, err := gitLabAdapter.Groups.GetGroup(args[0])
		doWePanic(err)

		for _, project := range group.Projects {
			renameDefault(project)
		}
	},
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

	viper.SetDefault("new-name", "main")
	viper.SetDefault("url", "https://gitlab.com")
	viper.SetDefault("unprotect", false)
	viper.SetDefault("delete", false)
	viper.SetDefault("devs-can-merge", false)
	viper.SetDefault("devs-can-push", false)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.grd.yaml)")
	rootCmd.Flags().StringP("new-name", "n", viper.GetString("new-name"), "Set the newname to rename the default to")
	rootCmd.Flags().StringP("token", "t", "", "GitLab Token (API access) to be used")
	rootCmd.Flags().StringP("url", "u", viper.GetString("url"), "GitLab API URL")
	rootCmd.Flags().BoolP("unprotect", "p", viper.GetBool("unprotect"), "Unprotect the old default branch when done")
	rootCmd.Flags().BoolP("delete", "d", viper.GetBool("delete"), "Delete the old default branch when done")
	rootCmd.Flags().Bool("devs-can-merge", viper.GetBool("devs-can-merge"), "For new protected branch: Are developers allowed to merge?")
	rootCmd.Flags().Bool("devs-can-push", viper.GetBool("devs-can-push"), "For new protected branch: Are developers allowed to push?")
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Print version information")

	viper.BindPFlag("new-name", rootCmd.Flags().Lookup("new-name"))
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))
	viper.BindPFlag("delete", rootCmd.Flags().Lookup("delete"))
	viper.BindPFlag("unprotect", rootCmd.Flags().Lookup("unprotect"))
	viper.BindPFlag("devs-can-merge", rootCmd.Flags().Lookup("devs-can-merge"))
	viper.BindPFlag("devs-can-push", rootCmd.Flags().Lookup("devs-can-push"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".grd")
	}
	viper.SetEnvPrefix("GRD")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func doWePanic(err error) {
	if err != nil {
		panic(err)
	}
}

func renameDefault(proj *gitlab.Project) {
	defaultBranchName := proj.DefaultBranch
	allBranches, _, err := gitLabAdapter.Branches.ListBranches(proj.ID, &gitlab.ListBranchesOptions{})
	doWePanic(err)
	if len(allBranches) > 0 {
		if defaultBranchName == viper.GetString("new-name") {
			fmt.Printf("[%s] Nothing todo, %s is already the default branch!\n", proj.Name, viper.GetString("new-name"))
		} else {
			fmt.Printf("[%s] Current defaultBranch is %s\n", proj.Name, defaultBranchName)

			createNewBranch(proj, allBranches, defaultBranchName)

			cleanupOldBranch(proj, allBranches, defaultBranchName)
		}
	} else {
		fmt.Printf("[%s] Doesn't contain any branches! Nothing to do.\n", proj.Name)
	}
	fmt.Println("----------------------------------------------------")
}

func branchAlreadyExists(branchName string, branches []*gitlab.Branch) (*gitlab.Branch, bool) {
	for _, branch := range branches {
		if branchName == branch.Name {
			return branch, true
		}
	}
	return nil, false
}

func createNewBranch(proj *gitlab.Project, allBranches []*gitlab.Branch, defaultBranchName string) {
	var err error
	_, newBranchExists := branchAlreadyExists(viper.GetString("new-name"), allBranches)
	if !newBranchExists {
		newBranch, _, err := gitLabAdapter.Branches.CreateBranch(proj.ID, &gitlab.CreateBranchOptions{
			Branch: gitlab.String(viper.GetString("new-name")),
			Ref:    gitlab.String(defaultBranchName),
		})
		doWePanic(err)
		fmt.Printf("[%s] Created new branch%s\n", proj.Name, newBranch.Name)
	}

	_, _, err = gitLabAdapter.Projects.EditProject(proj.ID, &gitlab.EditProjectOptions{
		DefaultBranch: gitlab.String(viper.GetString("new-name")),
	})
	doWePanic(err)
	fmt.Printf("[%s] Set default branch to %s\n", proj.Name, viper.GetString("new-name"))

	_, _, err = gitLabAdapter.Branches.ProtectBranch(proj.ID, viper.GetString("new-name"), &gitlab.ProtectBranchOptions{
		DevelopersCanPush:  gitlab.Bool(viper.GetBool("devs-can-push")),
		DevelopersCanMerge: gitlab.Bool(viper.GetBool("devs-can-merge")),
	})
	doWePanic(err)
	fmt.Printf("[%s] Protected branch %s\n", proj.Name, viper.GetString("new-name"))
}

func cleanupOldBranch(proj *gitlab.Project, allBranches []*gitlab.Branch, defaultBranchName string) {
	oldDefault, oldDefaultExists := branchAlreadyExists(defaultBranchName, allBranches)

	if viper.GetBool("unprotect") && oldDefault.Protected {
		branch, _, err := gitLabAdapter.Branches.UnprotectBranch(proj.ID, *gitlab.String(defaultBranchName))
		doWePanic(err)
		fmt.Printf("[%s] Unprotected branch %s\n", proj.Name, branch.Name)
	}

	if viper.GetBool("delete") && oldDefaultExists {
		if oldDefault.Protected {
			_, _, err := gitLabAdapter.Branches.UnprotectBranch(proj.ID, *gitlab.String(defaultBranchName))
			doWePanic(err)
		}
		_, err := gitLabAdapter.Branches.DeleteBranch(proj.ID, *gitlab.String(defaultBranchName))
		doWePanic(err)
		fmt.Printf("[%s] Deleted branch %s\n", proj.Name, defaultBranchName)
	}
}
