package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"

	"github.com/bigkevmcd/tekton-ci/pkg/githooks"
	"github.com/jenkins-x/go-scm/scm/factory"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

func makeHTTPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "execute PipelineRuns in response to hooks",
		Run: func(cmd *cobra.Command, args []string) {
			scmClient, err := factory.NewClient(viper.GetString("driver"), "", "")
			if err != nil {
				log.Fatal(err)
			}

			clusterConfig, err := rest.InClusterConfig()
			if err != nil {
				log.Fatalf("failed to get in cluster config: %v", err)
			}

			kubeClient, err := pipelineclientset.NewForConfig(clusterConfig)
			if err != nil {
				log.Fatalf("failed to get the versioned client: %v", err)
			}

			logger, _ := zap.NewProduction()
			defer logger.Sync() // flushes buffer, if any
			sugar := logger.Sugar()

			handler := githooks.New(
				http.DefaultClient,
				scmClient,
				kubeClient,
				viper.GetString("namespace"),
				sugar,
			)
			listen := fmt.Sprintf(":%d", viper.GetInt("port"))
			http.Handle("/", handler)
			log.Fatal(http.ListenAndServe(listen, nil))
		},
	}

	cmd.Flags().Int(
		"port",
		8080,
		"port to serve requests on",
	)
	logIfError(viper.BindPFlag("port", cmd.Flags().Lookup("port")))

	cmd.Flags().String(
		"driver",
		"github",
		"go-scm driver name to use e.g. github, gitlab",
	)
	logIfError(viper.BindPFlag("driver", cmd.Flags().Lookup("driver")))

	cmd.Flags().String(
		"namespace",
		"default",
		"namespace to execute PipelineRuns in",
	)
	logIfError(viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace")))
	return cmd
}
