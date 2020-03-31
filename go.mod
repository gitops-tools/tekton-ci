module github.com/bigkevmcd/tekton-ci

go 1.14

require (
	github.com/google/go-cmp v0.4.0
	github.com/jenkins-x/go-scm v1.5.83
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/tektoncd/pipeline v0.11.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	sigs.k8s.io/yaml v1.1.0
)

// Knative deps (release-0.12)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	knative.dev/caching => knative.dev/caching v0.0.0-20200116200605-67bca2c83dfa
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200113182502-b8dc5fbc6d2f
	knative.dev/pkg/vendor/github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
)
