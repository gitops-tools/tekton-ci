package dsl

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMakeVolumeClaimTemplate(t *testing.T) {
	wantedSz := resource.MustParse("1G")
	got := makeVolumeClaimTemplate(wantedSz)
	want := &corev1.PersistentVolumeClaim{
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
				Requests: corev1.ResourceList{"storage": wantedSz},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("makeVolumeClaimTemplate() failed: %s\n", diff)
	}
}
