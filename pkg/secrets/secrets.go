package secrets

import (
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeSecretGetter struct {
	coreClient kubernetes.Interface
	name       string
	namespace  string
}

// New creates and returns a KubeSecretGetter that looks up the hook secret as a
// key in a known v1.Secret.
func New(ns, n string, c kubernetes.Interface) *KubeSecretGetter {
	return &KubeSecretGetter{
		name:       n,
		namespace:  ns,
		coreClient: c,
	}
}

func (k KubeSecretGetter) Secret(hook scm.Webhook) (string, error) {
	secret, err := k.coreClient.CoreV1().Secrets(k.namespace).Get(k.name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	fullName := hook.Repository().FullName
	token, ok := secret.Data[fullName]
	if !ok {
		return "", fmt.Errorf("no secret for repository: %s", fullName)
	}

	return string(token), nil
}
