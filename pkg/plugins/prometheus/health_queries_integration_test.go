package prometheus

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

type promSeries struct {
	Series string `json:"series"`
	Values string `json:"values"`
}
type promSample struct {
	Labels string  `json:"labels"`
	Value  float64 `json:"value"`
}
type promExpression struct {
	Expr     string       `json:"expr"`
	EvalTime string       `json:"eval_time"`
	Samples  []promSample `json:"exp_samples"`
}
type promTest struct {
	Name        string           `json:"name"`
	Interval    string           `json:"interval"`
	Series      []promSeries     `json:"input_series"`
	Expressions []promExpression `json:"promql_expr_test"`
}

func healthQueryCases() []promTest {
	scope := `namespace="consize-demo",pod="checkout-api-abc-def"`
	container := scope + `,container="app"`
	info := promSeries{`kube_pod_container_info{` + container + `}`, "1+0x20"}
	periods := promSeries{`container_cpu_cfs_periods_total{` + container + `}`, "0+100x20"}
	idlePeriods := promSeries{periods.Series, "0+0x20"}
	throttled := promSeries{`container_cpu_cfs_throttled_periods_total{` + container + `}`, "0+0x20"}
	podInfo := promSeries{`kube_pod_info{` + scope + `}`, "1+0x20"}
	running := promSeries{`kube_pod_status_phase{` + scope + `,phase="Running"}`, "1+0x20"}
	failed := promSeries{`kube_pod_status_phase{` + scope + `,phase="Failed"}`, "1+0x20"}
	reason := promSeries{`kube_pod_status_reason{` + scope + `,reason="Evicted"}`, "1+0x20"}
	cases := []promTest{}
	add := func(name, query string, series []promSeries, value *float64) {
		samples := []promSample{}
		if value != nil {
			samples = append(samples, promSample{Labels: "{}", Value: *value})
		}
		cases = append(cases, promTest{Name: name, Interval: "30s", Series: series, Expressions: []promExpression{{Expr: renderQuery(query, "consize-demo", "checkout-api-.+"), EvalTime: "10m", Samples: samples}}})
	}
	zero, one, ratio := 0., 1., .2
	add("idle counters are observed zero", throttlingQuery(), []promSeries{info, idlePeriods, throttled}, &zero)
	add("active unthrottled", throttlingQuery(), []promSeries{info, periods, throttled}, &zero)
	add("active throttled", throttlingQuery(), []promSeries{info, periods, {throttled.Series, "0+20x20"}}, &ratio)
	add("missing numerator", throttlingQuery(), []promSeries{info, periods}, nil)
	add("missing denominator", throttlingQuery(), []promSeries{info, throttled}, nil)
	add("missing inventory", throttlingQuery(), []promSeries{periods, throttled}, nil)
	add("inconsistent counters", throttlingQuery(), []promSeries{info, idlePeriods, {throttled.Series, "0+20x20"}}, nil)
	add("partial container telemetry", throttlingQuery(), []promSeries{info, periods, throttled, {`kube_pod_container_info{` + scope + `,container="sidecar"}`, "1+0x20"}}, nil)
	add("sparse reason healthy pod", evictionQuery(), []promSeries{podInfo, running}, &zero)
	add("older exporter zero reason", evictionQuery(), []promSeries{podInfo, running, {reason.Series, "0+0x20"}}, &zero)
	add("evicted pod", evictionQuery(), []promSeries{podInfo, failed, reason}, &one)
	add("failed pod unknown reason", evictionQuery(), []promSeries{podInfo, failed}, nil)
	add("no state evidence", evictionQuery(), []promSeries{podInfo}, nil)
	add("missing pod inventory", evictionQuery(), []promSeries{running}, nil)
	add("partial pod state", evictionQuery(), []promSeries{podInfo, running, {`kube_pod_info{namespace="consize-demo",pod="checkout-api-xyz-abc"}`, "1+0x20"}}, nil)
	return cases
}

func TestLivePromQLHealthSemantics(t *testing.T) {
	if os.Getenv("CONSIZE_PROMQL_INTEGRATION") != "1" {
		t.Skip("set CONSIZE_PROMQL_INTEGRATION=1 to run official promtool in Docker Desktop")
	}
	fixture := struct {
		EvaluationInterval string     `json:"evaluation_interval"`
		Tests              []promTest `json:"tests"`
	}{"30s", healthQueryCases()}
	input, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "--context", "docker-desktop", "exec", "-i", "-n", "monitoring", "deployment/consize-prometheus-server", "-c", "prometheus-server", "--", "promtool", "test", "rules", "/dev/stdin")
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("promtool fixtures: %v\n%s", err, output)
	}
	t.Log(string(output))
}
