package secret

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates and returns a new Secret with the provided key in the data.
func Create(key string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tekton-ci-auth",
			Namespace: "testing",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			key: []byte(`secret-token`),
		},
	}
}
