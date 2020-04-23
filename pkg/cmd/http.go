package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/bigkevmcd/tekton-ci/pkg/dsl"
	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/pkg/metrics"
	"github.com/bigkevmcd/tekton-ci/pkg/secrets"
	"github.com/bigkevmcd/tekton-ci/pkg/spec"
	"github.com/bigkevmcd/tekton-ci/pkg/volumes"
)

const (
	defaultPipelineRunPrefix = "test-pipelinerun-"
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
				log.Fatalf("failed to create in-cluster config: %v", err)
			}

			tektonClient, err := pipelineclientset.NewForConfig(clusterConfig)
			if err != nil {
				log.Fatalf("failed to create the tekton client: %v", err)
			}

			coreClient, err := kubernetes.NewForConfig(clusterConfig)
			if err != nil {
				log.Fatalf("failed to create the core client: %v", err)
			}

			logger, _ := zap.NewProduction()
			defer func() {
				err := logger.Sync() // flushes buffer, if any
				if err != nil {
					log.Fatal(err)
				}
			}()
			sugar := logger.Sugar()

			namespace := viper.GetString("namespace")
			gitClient := git.New(scmClient, secrets.New(namespace, secrets.DefaultName, coreClient))
			dslHandler := dsl.New(
				gitClient,
				tektonClient,
				volumes.New(coreClient),
				metrics.New(nil),
				newDSLConfig(),
				namespace,
				sugar)
			specHandler := spec.New(
				gitClient,
				tektonClient,
				namespace,
				sugar)
			http.Handle("/pipeline", dslHandler)
			http.Handle("/pipelinerun", specHandler)
			http.Handle("/metrics", promhttp.Handler())
			listen := fmt.Sprintf(":%d", viper.GetInt("port"))
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

	bindConfigurationFlags(cmd)
	return cmd
}

func newDSLConfig() *dsl.Configuration {
	return &dsl.Configuration{
		ArchiverImage:             viper.GetString("archiver-image"),
		ArchiveURL:                viper.GetString("archive-url"),
		PipelineRunPrefix:         viper.GetString("pipelinerun-prefix"),
		DefaultServiceAccountName: viper.GetString("pipelinerun-serviceaccount-name"),
	}
}

func bindConfigurationFlags(cmd *cobra.Command) {
	cmd.Flags().String(
		"archiver-image",
		"",
		"image to execute for archiving artifacts",
	)
	logIfError(viper.BindPFlag("archiver-image", cmd.Flags().Lookup("archiver-image")))
	logIfError(cmd.MarkFlagRequired("archiver-image"))

	cmd.Flags().String(
		"archive-url",
		"",
		"passed to the archiver for configuration",
	)
	logIfError(viper.BindPFlag("archive-url", cmd.Flags().Lookup("archive-url")))
	logIfError(cmd.MarkFlagRequired("archive-url"))

	cmd.Flags().String(
		"pipelinerun-prefix",
		defaultPipelineRunPrefix,
		"used for the generateName in the generated PipelineRuns",
	)
	logIfError(viper.BindPFlag("pipelinerun-prefix", cmd.Flags().Lookup("pipelinerun-prefix")))

	cmd.Flags().String(
		"pipelinerun-serviceaccount-name",
		"default",
		"used for the generateName in the generated PipelineRuns",
	)
	logIfError(viper.BindPFlag("pipelinerun-serviceaccount-name", cmd.Flags().Lookup("pipelinerun-serviceaccount-name")))

}
