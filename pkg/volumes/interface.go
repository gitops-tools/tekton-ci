package volumes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Creator is an interface that defines the behaviour for creating new
// PersistentVolumeClaims with a a requisite size.
type Creator interface {
	Create(ctx context.Context, namespace string, size resource.Quantity) (*corev1.PersistentVolumeClaim, error)
}
