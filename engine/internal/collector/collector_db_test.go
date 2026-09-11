package collector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"consize/internal/dbmetrics"
	"consize/internal/store"
)

// TestCollectorIngestsDBSurface: with a DB source configured, Run
// upserts the demo instance as a Source="db" workload and fills
// usage_buckets with all five db_* metrics, step-aligned. The k8s
// surface stays untouched (empty fake prom + meta).
func TestCollectorIngestsDBSurface(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	fake := &fakeProm{cpu: matrixSeries(), mem: matrixSeries()}
	srv := newPromServer(fake)
	defer srv.Close()

	c := New(fakeMeta{}, NewHTTPPrometheus(srv.URL, nil), st, 15*time.Minute, time.Hour)
	c.DB = dbmetrics.NewFixture()
	if err := c.Run(ctx); err != nil {
		t.Fatal(err)
	}

	ws, err := st.ListWorkloads(ctx)
	if err != nil || len(ws) != 1 {
		t.Fatalf("want 1 workload (the db instance), got %d err=%v", len(ws), err)
	}
	wl := ws[0]
	if wl.Source != "db" || wl.Kind != "database" {
		t.Fatalf("db instance must be a Source=db workload: %+v", wl)
	}
	if wl.DBClass != "db.t3.large" || wl.DBReplicas != 1 ||
		wl.DBMaintenanceWindow == "" || wl.DBProvider != "fixture" {
		t.Fatalf("db fields: %+v", wl)
	}

	// All five db_* metrics present, step-aligned, with the instance id.
	for _, m := range []string{store.MetricDBCPUPercent, store.MetricDBIOPS,
		store.MetricDBConnections, store.MetricDBMemPercent, store.MetricDBErrors} {
		bs, err := st.ListBuckets(ctx, wl.ID, m, time.Now().Add(-2*time.Hour), time.Now())
		if err != nil || len(bs) == 0 {
			t.Fatalf("%s: want buckets, got %d err=%v", m, len(bs), err)
		}
		for _, b := range bs {
			if b.WorkloadID != wl.ID || b.WindowStart.Unix()%900 != 0 {
				t.Fatalf("%s bucket not aligned to instance: %+v", m, b)
			}
		}
	}

	// A second run must be idempotent: still one workload, and the
	// bucket set over a fixed past range is unchanged (upserts, not
	// duplicates). The range ends 10 minutes ago, truncated to the step
	// grid, so the moving "now" of each run cannot add a new boundary.
	rangeEnd := time.Now().Add(-10 * time.Minute).Truncate(15 * time.Minute)
	rangeStart := rangeEnd.Add(-time.Hour)
	before, _ := st.ListBuckets(ctx, wl.ID, store.MetricDBCPUPercent, rangeStart, rangeEnd)
	if err := c.Run(ctx); err != nil {
		t.Fatal(err)
	}
	ws, err = st.ListWorkloads(ctx)
	if err != nil || len(ws) != 1 {
		t.Fatalf("re-run must not duplicate workloads: %d err=%v", len(ws), err)
	}
	after, _ := st.ListBuckets(ctx, wl.ID, store.MetricDBCPUPercent, rangeStart, rangeEnd)
	if len(after) != len(before) || len(before) == 0 {
		t.Fatalf("re-run must be idempotent: %d → %d buckets", len(before), len(after))
	}
}

// TestCollectorPrometheusFailureStillIngestsDB: a Prometheus outage must not
// prevent the optional DB surface from running (issue #2).
func TestCollectorPrometheusFailureStillIngestsDB(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"status":"error","errorType":"bad_data","error":"parse error"}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(fakeMeta{}, NewHTTPPrometheus(srv.URL, nil), st, 15*time.Minute, time.Hour)
	c.DB = dbmetrics.NewFixture()
	if err := c.Run(ctx); err != nil {
		t.Fatalf("prometheus failure must not abort Run when DB is configured: %v", err)
	}

	ws, err := st.ListWorkloads(ctx)
	if err != nil || len(ws) != 1 {
		t.Fatalf("want 1 db workload despite prometheus failure, got %d err=%v", len(ws), err)
	}
	if ws[0].Source != "db" {
		t.Fatalf("expected Source=db workload, got %+v", ws[0])
	}
}
