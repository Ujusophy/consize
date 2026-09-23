package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDiscoverEmitsUniversalDeploymentResource(t *testing.T) {
	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "payments", UID: types.UID("uid-1"), ResourceVersion: "7", Labels: map[string]string{"consize.io/environment": "production", "consize.io/owner": "platform"}},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("512Mi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("1Gi")}}}}}}},
	}
	client := fake.NewSimpleClientset([]runtime.Object{deployment}...)
	p := NewWithPatcher(NewK8sPatcherFromClient(client))
	observations, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations = %d", len(observations))
	}
	got := observations[0].Resource
	if got.ID != "k8s:payments:checkout" || got.ProviderResourceID != "payments/checkout" {
		t.Fatalf("identity = %#v", got)
	}
	if got.Environment != "production" || got.Owner != "platform" {
		t.Fatalf("ownership = %#v", got)
	}
	if got.CurrentState["memory_request_bytes"] != int64(536870912) || got.CurrentState["cpu_request_millicores"] != int64(500) {
		t.Fatalf("state = %#v", got.CurrentState)
	}
}
