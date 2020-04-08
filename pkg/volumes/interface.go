package volumes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TODO: this should accept a resource.Quantity as the size, and probably
// extracted from
type Creator interface {
	Create(namespace string, size resource.Quantity) (*corev1.PersistentVolumeClaim, error)
}
