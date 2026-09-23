package kubernetes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func recoveryFixture(t *testing.T) (*Plugin, *fake.Clientset, plugin.ActionPlan) {
	t.Helper()
	ctx := context.Background()
	replicas := int32(1)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "test", UID: types.UID("immutable-uid"), Generation: 1}, Spec: appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{
		{Name: "app", Image: "example:v1", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceMemory: quantity.MustParse("512Mi"), corev1.ResourceCPU: quantity.MustParse("500m")}, Limits: corev1.ResourceList{corev1.ResourceMemory: quantity.MustParse("1Gi")}}},
		{Name: "sidecar", Image: "sidecar:v1", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceMemory: quantity.MustParse("128Mi")}, Limits: corev1.ResourceList{corev1.ResourceMemory: quantity.MustParse("256Mi")}}},
	}}}}, Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, ReadyReplicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1}}
	client := fake.NewSimpleClientset(dep)
	p := NewWithPatcher(NewK8sPatcherFromClient(client))
	plan, err := p.Plan(ctx, plugin.ActionInput{Resource: resource.Resource{ID: "app", Type: resource.TypeKubernetesDeployment, Metadata: map[string]any{"namespace": "test", "name": "app"}, CurrentState: map[string]any{"memory_request_bytes": int64(640 * 1024 * 1024), "memory_limit_bytes": int64(1280 * 1024 * 1024)}}, Parameters: map[string]any{"patch": PatchDiff{Resource: "memory", CurrentReq: 640 * 1024 * 1024, ProposedReq: 480 * 1024 * 1024, CurrentLimit: 1280 * 1024 * 1024, ProposedLimit: 1280 * 1024 * 1024}}})
	if err != nil {
		t.Fatal(err)
	}
	return p, client, plan
}

func TestSnapshotApplyAndRollbackAreIdempotentAfterSerialization(t *testing.T) {
	p, client, plan := recoveryFixture(t)
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := p.Execute(ctx, plan); err != nil {
			t.Fatal(err)
		}
	}
	state, err := p.Inspect(ctx, plan)
	if err != nil || state != plugin.StateApplied {
		t.Fatalf("state=%s error=%v", state, err)
	}
	for i := 0; i < 2; i++ {
		if _, err := p.Rollback(ctx, plan); err != nil {
			t.Fatal(err)
		}
	}
	state, err = p.Inspect(ctx, plan)
	if err != nil || state != plugin.StateOriginal {
		t.Fatal("original state not restored")
	}
	dep, _ := client.AppsV1().Deployments("test").Get(ctx, "app", metav1.GetOptions{})
	if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().Value() != 512*1024*1024 || dep.Spec.Template.Spec.Containers[1].Resources.Requests.Memory().Value() != 128*1024*1024 {
		t.Fatal("exact per-container state not restored")
	}
	if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() != 500 {
		t.Fatal("unrelated CPU changed")
	}
}

func TestSnapshotRejectsWorkloadVersionAndIdentityDrift(t *testing.T) {
	for _, change := range []string{"image", "uid", "resources"} {
		t.Run(change, func(t *testing.T) {
			p, client, plan := recoveryFixture(t)
			ctx := context.Background()
			if _, err := p.Execute(ctx, plan); err != nil {
				t.Fatal(err)
			}
			dep, _ := client.AppsV1().Deployments("test").Get(ctx, "app", metav1.GetOptions{})
			switch change {
			case "image":
				dep.Spec.Template.Spec.Containers[0].Image = "example:v2"
			case "uid":
				dep.UID = "replacement"
			case "resources":
				dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = quantity.MustParse("256Mi")
			}
			if _, err := client.AppsV1().Deployments("test").Update(ctx, dep, metav1.UpdateOptions{}); err != nil {
				t.Fatal(err)
			}
			if state, err := p.Inspect(ctx, plan); err != nil || state != plugin.StateDrifted {
				t.Fatal("drift not detected")
			}
			if _, err := p.Rollback(ctx, plan); err == nil {
				t.Fatal("drift overwritten")
			}
		})
	}
}
