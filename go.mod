module github.com/bigkevmcd/tekton-ci

go 1.14

require (
	github.com/google/go-cmp v0.4.0
	github.com/jenkins-x/go-scm v1.5.83
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/tektoncd/pipeline v0.10.2
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.1
	sigs.k8s.io/yaml v1.1.0
)

// Pin k8s deps to 1.13.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
)
