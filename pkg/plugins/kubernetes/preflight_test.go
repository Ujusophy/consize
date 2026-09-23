package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func checkStatus(t *testing.T, checks []plugin.PreflightCheck, id string) string {
	t.Helper()
	for _, c := range checks {
		if c.ID == id {
			return c.Status
		}
	}
	t.Fatalf("missing %s check: %+v", id, checks)
	return ""
}
func utilizationHPA(name string, key corev1.ResourceName) *autoscalingv2.HorizontalPodAutoscaler {
	percentage := int32(70)
	return &autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"}, Spec: autoscalingv2.HorizontalPodAutoscalerSpec{ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "app"}, Metrics: []autoscalingv2.MetricSpec{{Type: autoscalingv2.ResourceMetricSourceType, Resource: &autoscalingv2.ResourceMetricSource{Name: key, Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: &percentage}}}}}}
}

func TestHPAResourceAndContainerConflicts(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		mutate     func(*autoscalingv2.HorizontalPodAutoscaler)
	}{
		{"memory utilization", "blocked", func(h *autoscalingv2.HorizontalPodAutoscaler) {}},
		{"unaffected CPU", "passed", func(h *autoscalingv2.HorizontalPodAutoscaler) { h.Spec.Metrics[0].Resource.Name = corev1.ResourceCPU }},
		{"absolute memory", "passed", func(h *autoscalingv2.HorizontalPodAutoscaler) {
			q := quantity.MustParse("300Mi")
			h.Spec.Metrics[0].Resource.Target = autoscalingv2.MetricTarget{Type: autoscalingv2.AverageValueMetricType, AverageValue: &q}
		}},
		{"different target", "passed", func(h *autoscalingv2.HorizontalPodAutoscaler) { h.Spec.ScaleTargetRef.Name = "another-app" }},
		{"container memory utilization", "blocked", func(h *autoscalingv2.HorizontalPodAutoscaler) {
			target := h.Spec.Metrics[0].Resource.Target
			h.Spec.Metrics = []autoscalingv2.MetricSpec{{Type: autoscalingv2.ContainerResourceMetricSourceType, ContainerResource: &autoscalingv2.ContainerResourceMetricSource{Name: corev1.ResourceMemory, Container: "app", Target: target}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, client, plan := recoveryFixture(t)
			h := utilizationHPA("scale", corev1.ResourceMemory)
			tc.mutate(h)
			if _, err := client.AutoscalingV2().HorizontalPodAutoscalers("test").Create(context.Background(), h, metav1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
			checks, err := p.Preflight(context.Background(), plan)
			if err != nil {
				t.Fatal(err)
			}
			if got := checkStatus(t, checks, "hpa"); got != tc.want {
				t.Fatalf("got %s: %+v", got, checks)
			}
		})
	}
}

func TestControllerAppearingAfterReviewBlocksExecution(t *testing.T) {
	p, client, plan := recoveryFixture(t)
	ctx := context.Background()
	if err := plugin.RequirePreflight(plan.Preflight); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AutoscalingV2().HorizontalPodAutoscalers("test").Create(ctx, utilizationHPA("late-hpa", corev1.ResourceMemory), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := p.Execute(ctx, plan)
	var blocked *plugin.PreflightError
	if !errors.As(err, &blocked) {
		t.Fatalf("late controller was not blocked: %v", err)
	}
	if state, err := p.Inspect(ctx, plan); err != nil || state != plugin.StateOriginal {
		t.Fatal("Deployment mutated despite conflict")
	}
}

func TestUnknownInspectionBlocksMutation(t *testing.T) {
	p, client, plan := recoveryFixture(t)
	client.PrependReactor("list", "horizontalpodautoscalers", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "autoscaling", Resource: "horizontalpodautoscalers"}, "", errors.New("denied"))
	})
	checks, err := p.Preflight(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if checkStatus(t, checks, "hpa") != "unknown" {
		t.Fatal("permission denial treated as absent controller")
	}
	if _, err := p.Execute(context.Background(), plan); err == nil {
		t.Fatal("mutation allowed with denied inspection")
	}
}

func TestQoSDowngradeIsReviewableButNotExecutable(t *testing.T) {
	p, client, _ := recoveryFixture(t)
	ctx := context.Background()
	dep, err := client.AppsV1().Deployments("test").Get(ctx, "app", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	dep.Spec.Template.Spec.Containers = dep.Spec.Template.Spec.Containers[:1]
	r := &dep.Spec.Template.Spec.Containers[0].Resources
	r.Limits[corev1.ResourceCPU] = quantity.MustParse("500m")
	r.Limits[corev1.ResourceMemory] = quantity.MustParse("512Mi")
	if _, err = client.AppsV1().Deployments("test").Update(ctx, dep, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	plan, err := p.Plan(ctx, plugin.ActionInput{Resource: resource.Resource{ID: "app", Type: resource.TypeKubernetesDeployment, Metadata: map[string]any{"namespace": "test", "name": "app"}}, Parameters: map[string]any{"patch": PatchDiff{Resource: "memory", CurrentReq: 512 * 1024 * 1024, ProposedReq: 384 * 1024 * 1024, CurrentLimit: 512 * 1024 * 1024, ProposedLimit: 512 * 1024 * 1024}}})
	if err != nil {
		t.Fatal("blocked change must remain reviewable", err)
	}
	if checkStatus(t, plan.Preflight, "qos") != "blocked" {
		t.Fatalf("QoS downgrade not reported: %+v", plan.Preflight)
	}
	if _, err = p.Execute(ctx, plan); err == nil {
		t.Fatal("Guaranteed downgrade allowed")
	}
}

func TestQoSIncludesInitContainersAndRejectsPodBudgets(t *testing.T) {
	r := corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: quantity.MustParse("1"), corev1.ResourceMemory: quantity.MustParse("1Gi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: quantity.MustParse("1000m"), corev1.ResourceMemory: quantity.MustParse("1024Mi")}}
	spec := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Resources: r}}}
	if q, err := qosClass(spec); err != nil || q != corev1.PodQOSGuaranteed {
		t.Fatalf("equivalent quantities: %s %v", q, err)
	}
	spec.InitContainers = []corev1.Container{{Name: "init"}}
	if q, err := qosClass(spec); err != nil || q != corev1.PodQOSBurstable {
		t.Fatal("init container ignored")
	}
	spec.Resources = &r
	if _, err := qosClass(spec); err == nil {
		t.Fatal("unsupported pod-level QoS guessed")
	}
}

func TestVPAModesAndMissingPermissions(t *testing.T) {
	for _, mode := range []string{"Off", "Initial", "Recreate", "InPlaceOrRecreate", "InPlace", "Auto", "", "Unknown"} {
		t.Run(mode, func(t *testing.T) {
			p, client, plan := recoveryFixture(t)
			gvr := schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}
			client.Discovery().(*discoveryfake.FakeDiscovery).Resources = []*metav1.APIResourceList{{GroupVersion: "autoscaling.k8s.io/v1", APIResources: []metav1.APIResource{{Name: gvr.Resource, Kind: "VerticalPodAutoscaler", Namespaced: true}}}}
			obj := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "autoscaling.k8s.io/v1", "kind": "VerticalPodAutoscaler", "metadata": map[string]any{"name": "manage", "namespace": "test"}, "spec": map[string]any{"targetRef": map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "name": "app"}}}}
			if mode != "" {
				obj.Object["spec"].(map[string]any)["updatePolicy"] = map[string]any{"updateMode": mode}
			}
			dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{gvr: "VerticalPodAutoscalerList"}, obj)
			p.patcher = NewK8sPatcherFromClients(client, dynamicClient)
			checks, err := p.Preflight(context.Background(), plan)
			if err != nil {
				t.Fatal(err)
			}
			want := "blocked"
			if mode == "Off" {
				want = "passed"
			}
			if mode == "Unknown" {
				want = "unknown"
			}
			if got := checkStatus(t, checks, "vpa"); got != want {
				t.Fatalf("mode %s: %+v", mode, checks)
			}
			dynamicClient.PrependReactor("list", gvr.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(gvr.GroupResource(), "", errors.New("denied"))
			})
			checks, err = p.Preflight(context.Background(), plan)
			if err != nil || checkStatus(t, checks, "vpa") != "unknown" {
				t.Fatal("VPA permission denial not fail-closed")
			}
		})
	}
}

func TestRollbackNotBlockedByNewController(t *testing.T) {
	p, client, plan := recoveryFixture(t)
	ctx := context.Background()
	if _, err := p.Execute(ctx, plan); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AutoscalingV2().HorizontalPodAutoscalers("test").Create(ctx, utilizationHPA("late-hpa", corev1.ResourceMemory), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Rollback(ctx, plan); err != nil {
		t.Fatal("forward guard prevented restoration", err)
	}
	if state, err := p.Inspect(ctx, plan); err != nil || state != plugin.StateOriginal {
		t.Fatal("original resources not restored")
	}
}
