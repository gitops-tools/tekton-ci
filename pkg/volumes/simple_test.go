package volumes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ Creator = (*SimpleVolumeCreator)(nil)

func TestSimpleVolume(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	c := New(fakeClient)
	size := resource.MustParse("1Gi")
	v, err := c.Create("testing", size)
	if err != nil {
		t.Fatal(err)
	}

	want := &corev1.PersistentVolumeClaim{
		TypeMeta: volumeTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namePrefix,
			Namespace:    "testing",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": size,
				},
			},
			VolumeMode: &SimpleVolumeMode,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
		},
	}
	if diff := cmp.Diff(want, v); diff != "" {
		t.Fatalf("new volume failed: %s\n", diff)
	}

	created, err := fakeClient.CoreV1().PersistentVolumeClaims("testing").Get("", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Fatalf("saved volume was different: %s\n", diff)
	}
}
