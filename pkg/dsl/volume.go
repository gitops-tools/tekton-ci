package dsl

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var pvcMeta = metav1.TypeMeta{
	APIVersion: "v1",
	Kind:       "PersistentVolumeClaim",
}

func strPtr(s string) *string {
	return &s
}

func makeVolumeClaimTemplate(sz resource.Quantity) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: pvcMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pipeline-run-pvc-",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: strPtr("manual"),
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"storage": sz},
			},
		},
	}
}
