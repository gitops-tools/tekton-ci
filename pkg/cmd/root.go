package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
		Use:   "ci-hook-server",
		Short: "Tekton CI",
		Run: func(cmd *cobra.Command, args []string) {
			f, err := os.Open(viper.GetString("pipeline-file"))
			logIfError(err)
			defer f.Close()

			parsed, err := ci.Parse(f)
			logIfError(err)
			source := &dsl.Source{RepoURL: viper.GetString("repository-url"), Ref: viper.GetString("branch")}

			logger, _ := zap.NewProduction()
			defer func() {
				logIfError(logger.Sync()) // flushes buffer, if any
			}()
			sugar := logger.Sugar()

			converted, err := dsl.Convert(parsed, sugar, newDSLConfig(), source, "shared-task-storage", nil, "unique-id")
			if err != nil {
				log.Fatalf("error converting the script: %v", err)
			}

			d, err := yaml.Marshal(converted)
			if err != nil {
				log.Fatalf("error marshaling YAML: %v", err)
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
	bindConfigurationFlags(cmd)
	cmd.AddCommand(makeHTTPCmd())
	return cmd
}

func initConfig() {
	viper.AutomaticEnv()
}

// Execute is the main entry point into this component.
func Execute() {
	if err := makeRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
