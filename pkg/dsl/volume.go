package dsl

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/storage/names"
)

const pvcPrefix = "pipeline-run-pvc-"

var pvcMeta = metav1.TypeMeta{
	APIVersion: "v1",
	Kind:       "PersistentVolumeClaim",
}

var nameGenerator = func() string {
	return names.SimpleNameGenerator.GenerateName(pvcPrefix)
}

func makeVolumeClaimTemplate(sz resource.Quantity) *corev1.PersistentVolumeClaim {
	simpleVolumeMode := corev1.PersistentVolumeFilesystem
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: nameGenerator(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeMode: &simpleVolumeMode,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"storage": sz},
			},
		},
	}
}

func strPtr(s string) *string {
	return &s
}
