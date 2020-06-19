package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"

	"github.com/gitops-tools/tekton-ci/pkg/cel"
	"github.com/gitops-tools/tekton-ci/pkg/ci"
	"github.com/gitops-tools/tekton-ci/pkg/dsl"
)

func makeConvertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert --pipeline-file",
		Short: "convert",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := os.Open(viper.GetString("pipeline-file"))
			if err != nil {
				return err
			}
			defer f.Close()

			parsed, err := ci.Parse(f)
			if err != nil {
				return err
			}
			source := &dsl.Source{RepoURL: viper.GetString("repository-url"), Ref: viper.GetString("branch")}

			logger, _ := zap.NewProduction()
			sugar := logger.Sugar()

			fakeHook := map[string]interface{}{}
			ctx, err := cel.New(fakeHook)
			if err != nil {
				return err
			}
			converted, err := dsl.Convert(parsed, sugar, newDSLConfig(), source, "shared-task-storage", ctx, "unique-id")
			if err != nil {
				return err
			}

			d, err := yaml.Marshal(converted)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", string(d))
			return nil
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
	return cmd
}
