package volumes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	namePrefix = "simple-volume-"
)

var (
	volumeTypeMeta = metav1.TypeMeta{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	}

	volumeMode = corev1.PersistentVolumeFilesystem
)

// New creates and returns a VolumeCreator that creates fixed size, Filesystem
// based PersistentVolumeClaims.
func New(c kubernetes.Interface) *SimpleVolumeCreator {
	return &SimpleVolumeCreator{
		coreClient: c,
	}
}

// SimpleVolumeCreator is an implementation of the Creator interface.
type SimpleVolumeCreator struct {
	coreClient kubernetes.Interface
}

// Create impements the Creator interface.
func (s SimpleVolumeCreator) Create(namespace string, size resource.Quantity) (*corev1.PersistentVolumeClaim, error) {
	vc := &corev1.PersistentVolumeClaim{
		TypeMeta: volumeTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namePrefix,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": size,
				},
			},
			VolumeMode: &volumeMode,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
		},
	}
	volume, err := s.coreClient.CoreV1().PersistentVolumeClaims(namespace).Create(vc)
	if err != nil {
		return nil, err
	}
	return volume, nil
}
