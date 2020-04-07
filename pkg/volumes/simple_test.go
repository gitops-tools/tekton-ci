package volumes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ Creator = (*SimpleVolumeCreator)(nil)

func TestSimpleVolume(t *testing.T) {
	c := New()
	size := resource.MustParse("1Gi")
	v, err := c.Create(size)
	if err != nil {
		t.Fatal(err)
	}

	want := &corev1.PersistentVolumeClaim{
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
	if diff := cmp.Diff(want, v); diff != "" {
		t.Fatalf("new volume failed: %s\n", diff)
	}
}
