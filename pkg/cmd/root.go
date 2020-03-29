package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/pipelines"
)

func init() {
	rootCmd.Flags().String(
		"pipeline-file",
		"",
		"YAML with pipeline description",
	)
	rootCmd.MarkFlagRequired("pipeline-file")
	viper.BindPFlag("pipeline-file", rootCmd.Flags().Lookup("pipeline-file"))
	rootCmd.Flags().String(
		"pipelinerun-name",
		"test-pipelinerun",
		"inserted into the generated PipelineRun resource",
	)
	viper.BindPFlag("pipelinerun-name", rootCmd.Flags().Lookup("pipelinerun-name"))
	rootCmd.Flags().String(
		"repository-url",
		"",
		"e.g. https://github.com/my-org/my-repo.git",
	)
	viper.BindPFlag("repository-url", rootCmd.Flags().Lookup("repository-url"))
	rootCmd.MarkFlagRequired("repository-url")
	rootCmd.Flags().String(
		"branch",
		"master",
		"checkout and execute against this branch",
	)
	viper.BindPFlag("branch", rootCmd.Flags().Lookup("branch"))
	rootCmd.MarkFlagRequired("branch")

	cobra.OnInitialize(initConfig)
}

var rootCmd = &cobra.Command{
	Use:   "testing",
	Short: "Generate a TektonCD PipelineRun from a CI pipeline description",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := os.Open(viper.GetString("pipeline-file"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		parsed, err := ci.Parse(f)
		if err != nil {
			log.Fatal(err)
		}
		source := &pipelines.Source{RepoURL: viper.GetString("repository-url"), Ref: viper.GetString("branch")}
		converted := pipelines.Convert(parsed, viper.GetString("pipelinerun-name"), source)

		d, err := yaml.Marshal(converted)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("%s\n", string(d))
	},
}

func initConfig() {
	viper.AutomaticEnv()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
