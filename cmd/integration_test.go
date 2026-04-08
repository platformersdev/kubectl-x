//go:build integration

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func podListResponse(names ...string) map[string]interface{} {
	items := make([]interface{}, 0, len(names))
	for _, name := range names {
		items = append(items, map[string]interface{}{
			"kind":       "Pod",
			"apiVersion": "v1",
			"metadata":   map[string]interface{}{"name": name, "namespace": "default"},
			"status":     map[string]interface{}{"phase": "Running"},
		})
	}
	return map[string]interface{}{
		"kind":       "PodList",
		"apiVersion": "v1",
		"metadata":   map[string]interface{}{"resourceVersion": "1"},
		"items":      items,
	}
}

// TestGetPodsJSON verifies that pod lists from multiple contexts are merged into
// a single JSON List and that each item carries a metadata.context annotation.
func TestGetPodsJSON(t *testing.T) {
	h := NewHarness(t)
	s1 := h.AddContext("ctx1")
	s2 := h.AddContext("ctx2")

	s1.HandleJSON("/api/v1/namespaces/default/pods", podListResponse("pod-a", "pod-b"))
	s2.HandleJSON("/api/v1/namespaces/default/pods", podListResponse("pod-c"))

	out, err := h.Run("get", "pods", "-o", "json")
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)

	contextsSeen := make(map[string]bool)
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		require.True(t, ok)
		meta, ok := m["metadata"].(map[string]interface{})
		require.True(t, ok)
		ctx, _ := meta["context"].(string)
		contextsSeen[ctx] = true
	}
	assert.True(t, contextsSeen["ctx1"], "expected items from ctx1")
	assert.True(t, contextsSeen["ctx2"], "expected items from ctx2")
}

// TestVersion verifies that version output includes context names and the
// server version returned by the fake API.
func TestVersion(t *testing.T) {
	h := NewHarness(t)
	h.AddContext("ctx1")
	h.AddContext("ctx2")

	out, err := h.Run("version")
	require.NoError(t, err)

	assert.Contains(t, out, "CONTEXT")
	assert.Contains(t, out, "SERVER VERSION")
	assert.Contains(t, out, "ctx1")
	assert.Contains(t, out, "ctx2")
	assert.Contains(t, out, "v1.28.0")
}

// TestExcludeContext verifies that --exclude filters out matching contexts.
func TestExcludeContext(t *testing.T) {
	h := NewHarness(t)
	s1 := h.AddContext("prod-east")
	s2 := h.AddContext("dev-west")

	s1.HandleJSON("/api/v1/namespaces/default/pods", podListResponse("prod-pod"))
	s2.HandleJSON("/api/v1/namespaces/default/pods", podListResponse("dev-pod"))

	excludePatterns = []string{"dev"}
	t.Cleanup(func() { excludePatterns = []string{} })

	out, err := h.Run("get", "pods", "-o", "json")
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	m := items[0].(map[string]interface{})
	meta := m["metadata"].(map[string]interface{})
	assert.Equal(t, "prod-east", meta["context"])
}

// TestIncludeContext verifies that --include limits results to matching contexts.
func TestIncludeContext(t *testing.T) {
	h := NewHarness(t)
	s1 := h.AddContext("prod-east")
	s2 := h.AddContext("dev-west")

	s1.HandleJSON("/api/v1/namespaces/default/pods", podListResponse("prod-pod"))
	s2.HandleJSON("/api/v1/namespaces/default/pods", podListResponse("dev-pod"))

	filterPatterns = []string{"prod"}
	t.Cleanup(func() { filterPatterns = []string{} })

	out, err := h.Run("get", "pods", "-o", "json")
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	m := items[0].(map[string]interface{})
	meta := m["metadata"].(map[string]interface{})
	assert.Equal(t, "prod-east", meta["context"])
}
