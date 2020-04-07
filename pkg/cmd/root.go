package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/dsl"
)

func init() {
	cobra.OnInitialize(initConfig)
}

func logIfError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func makeRootCmd() *cobra.Command {
	cmd := &cobra.Command{
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
			source := &dsl.Source{RepoURL: viper.GetString("repository-url"), Ref: viper.GetString("branch")}
			converted := dsl.Convert(parsed, viper.GetString("pipelinerun-name"), source)

			d, err := yaml.Marshal(converted)
			if err != nil {
				log.Fatalf("error: %v", err)
			}
			fmt.Printf("%s\n", string(d))
		},
	}
	cmd.Flags().String(
		"pipeline-file",
		"",
		"YAML with pipeline description",
	)
	logIfError(cmd.MarkFlagRequired("pipeline-file"))
	logIfError(viper.BindPFlag("pipeline-file", cmd.Flags().Lookup("pipeline-file")))
	cmd.Flags().String(
		"pipelinerun-name",
		"test-pipelinerun",
		"inserted into the generated PipelineRun resource",
	)
	logIfError(viper.BindPFlag("pipelinerun-name", cmd.Flags().Lookup("pipelinerun-name")))
	cmd.Flags().String(
		"repository-url",
		"",
		"e.g. https://github.com/my-org/my-repo.git",
	)
	logIfError(viper.BindPFlag("repository-url", cmd.Flags().Lookup("repository-url")))
	logIfError(cmd.MarkFlagRequired("repository-url"))
	cmd.Flags().String(
		"branch",
		"master",
		"checkout and execute against this branch",
	)
	logIfError(viper.BindPFlag("branch", cmd.Flags().Lookup("branch")))
	logIfError(cmd.MarkFlagRequired("branch"))
	cmd.AddCommand(makeHTTPCmd())
	return cmd
}

func initConfig() {
	viper.AutomaticEnv()
}

func Execute() {
	if err := makeRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
