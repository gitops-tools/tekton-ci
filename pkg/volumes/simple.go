package volumes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// New creates and returns a volume creator that creates volumes with a fixed
// size.
func New() *SimpleVolumeCreator {
	return &SimpleVolumeCreator{}
}

type SimpleVolumeCreator struct {
}

func (s SimpleVolumeCreator) Create(size resource.Quantity) (*corev1.PersistentVolumeClaim, error) {
	return &corev1.PersistentVolumeClaim{
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
	}, nil
}
