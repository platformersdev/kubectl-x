package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// FakeServer is a minimal fake Kubernetes API server for use in integration tests.
// It registers sensible defaults for API discovery endpoints and lets tests
// register additional handlers for specific paths via HandleJSON.
type FakeServer struct {
	Server *httptest.Server
	mux    *http.ServeMux
}

func newFakeServer(t *testing.T) *FakeServer {
	t.Helper()
	mux := http.NewServeMux()
	fs := &FakeServer{mux: mux}
	fs.registerDefaults()
	fs.Server = httptest.NewServer(mux)
	t.Cleanup(fs.Server.Close)
	return fs
}

// HandleJSON registers a handler for path that encodes v as a JSON response.
// Calling HandleJSON again for the same path replaces the previous handler.
func (fs *FakeServer) HandleJSON(path string, v interface{}) {
	fs.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (fs *FakeServer) registerDefaults() {
	fs.HandleJSON("/api", map[string]interface{}{
		"kind":       "APIVersions",
		"apiVersion": "v1",
		"versions":   []string{"v1"},
		"serverAddressByClientCIDRs": []map[string]interface{}{
			{"clientCIDR": "0.0.0.0/0", "serverAddress": ""},
		},
	})
	fs.HandleJSON("/apis", map[string]interface{}{
		"kind":       "APIGroupList",
		"apiVersion": "v1",
		"groups":     []interface{}{},
	})
	fs.HandleJSON("/api/v1", map[string]interface{}{
		"kind":         "APIResourceList",
		"groupVersion": "v1",
		"resources": []map[string]interface{}{
			{
				"name":       "pods",
				"namespaced": true,
				"kind":       "Pod",
				"verbs":      []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				"name":       "nodes",
				"namespaced": false,
				"kind":       "Node",
				"verbs":      []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
		},
	})
	fs.HandleJSON("/version", map[string]interface{}{
		"major":        "1",
		"minor":        "28",
		"gitVersion":   "v1.28.0",
		"gitCommit":    "abc1234",
		"gitTreeState": "clean",
		"buildDate":    "2023-08-15T10:20:12Z",
		"goVersion":    "go1.20.7",
		"compiler":     "gc",
		"platform":     "linux/amd64",
	})
}

// Harness manages a set of fake servers and a temporary kubeconfig for integration testing.
type Harness struct {
	t              *testing.T
	servers        map[string]*FakeServer
	contextOrder   []string
	kubeconfigPath string
	tmpDir         string
}

// NewHarness creates a new Harness backed by a temporary directory that is
// cleaned up when the test ends.
func NewHarness(t *testing.T) *Harness {
	t.Helper()
	tmpDir := t.TempDir()
	return &Harness{
		t:              t,
		servers:        make(map[string]*FakeServer),
		kubeconfigPath: filepath.Join(tmpDir, "kubeconfig"),
		tmpDir:         tmpDir,
	}
}

// AddContext creates a fake server, registers it as a kubeconfig context with
// the given name, and returns the server so the test can register handlers.
func (h *Harness) AddContext(name string) *FakeServer {
	h.t.Helper()
	fs := newFakeServer(h.t)
	h.servers[name] = fs
	h.contextOrder = append(h.contextOrder, name)
	h.writeKubeconfig()
	return fs
}

func (h *Harness) writeKubeconfig() {
	h.t.Helper()

	var clusters, contexts, users []interface{}
	for _, name := range h.contextOrder {
		fs := h.servers[name]
		clusters = append(clusters, map[string]interface{}{
			"name": name,
			"cluster": map[string]interface{}{
				"server": fs.Server.URL,
			},
		})
		contexts = append(contexts, map[string]interface{}{
			"name": name,
			"context": map[string]interface{}{
				"cluster": name,
				"user":    name,
			},
		})
		users = append(users, map[string]interface{}{
			"name": name,
			"user": map[string]interface{}{},
		})
	}

	kubeconfig := map[string]interface{}{
		"apiVersion":      "v1",
		"kind":            "Config",
		"clusters":        clusters,
		"contexts":        contexts,
		"users":           users,
		"current-context": h.contextOrder[0],
	}

	data, err := yaml.Marshal(kubeconfig)
	require.NoError(h.t, err)
	require.NoError(h.t, os.WriteFile(h.kubeconfigPath, data, 0600))
}

// Run executes a kubectl-x command and returns the captured stdout.
// KUBECONFIG is pointed at the harness kubeconfig and HOME is set to a temp
// dir so kubectl's discovery cache is isolated from the real home directory.
func (h *Harness) Run(subcommand string, args ...string) (string, error) {
	h.t.Helper()

	h.t.Setenv("KUBECONFIG", h.kubeconfigPath)
	h.t.Setenv("HOME", h.tmpDir) // isolate kubectl's cache

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(h.t, err)
	os.Stdout = w

	var buf bytes.Buffer
	readerDone := make(chan struct{})
	go func() {
		io.Copy(&buf, r)
		close(readerDone)
	}()

	runErr := runCommand(subcommand, args)

	w.Close()
	<-readerDone
	os.Stdout = oldStdout
	r.Close()

	return buf.String(), runErr
}
