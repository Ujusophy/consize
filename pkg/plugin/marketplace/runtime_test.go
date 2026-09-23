package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/consize-oss/consize/pkg/plugin"
)

type testPlugin struct{}

func (testPlugin) ID() string { return "test" }
func (testPlugin) Manifest() plugin.Manifest {
	return plugin.Manifest{ID: "test", Category: plugin.CategoryMetrics}
}
func (testPlugin) Health(context.Context) plugin.Health { return plugin.Health{Status: "healthy"} }
func TestProtocolDispatch(t *testing.T) {
	for _, tc := range []struct {
		method   string
		protocol int
		fails    bool
	}{{"handshake", Protocol, false}, {"health", Protocol, false}, {"execute", Protocol, true}, {"unknown", Protocol, true}, {"health", 999, true}} {
		var out bytes.Buffer
		b, _ := json.Marshal(Request{Protocol: tc.protocol, Method: tc.method})
		err := ServeOnce(context.Background(), bytes.NewReader(b), &out, func(json.RawMessage) (plugin.Plugin, error) { return testPlugin{}, nil })
		if err != nil {
			t.Fatal(err)
		}
		var response Response
		if err = json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if (response.Error != "") != tc.fails {
			t.Fatalf("%s: %+v", tc.method, response)
		}
	}
}
func TestProtocolRejectsMultipleRequests(t *testing.T) {
	var out bytes.Buffer
	if err := ServeOnce(context.Background(), bytes.NewBufferString(`{"protocol":1,"method":"health"} {}`), &out, func(json.RawMessage) (plugin.Plugin, error) { return testPlugin{}, nil }); err == nil {
		t.Fatal("multiple requests accepted")
	}
}
func TestBoundedOutput(t *testing.T) {
	b := &limitedBuffer{limit: 3}
	if _, err := b.Write([]byte("oversize")); err == nil {
		t.Fatal("unbounded output")
	}
}
