package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeMinimalKubeconfig(t *testing.T, contextNames []string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := tmpDir + "/kubeconfig"

	var contexts []map[string]interface{}
	for _, name := range contextNames {
		contexts = append(contexts, map[string]interface{}{"name": name})
	}
	data, err := yaml.Marshal(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Config",
		"contexts":   contexts,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
	return path
}

func captureList(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	runErr := runList()
	w.Close()
	<-done
	os.Stdout = old
	r.Close()
	return buf.String(), runErr
}

func TestRunList(t *testing.T) {
	t.Run("lists all contexts", func(t *testing.T) {
		path := writeMinimalKubeconfig(t, []string{"dev-use1-gkg2", "prod-usw2-ejlr", "prod-use1-arj3"})
		t.Setenv("KUBECONFIG", path)

		out, err := captureList(t)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(out), "\n")
		assert.Equal(t, []string{"dev-use1-gkg2", "prod-usw2-ejlr", "prod-use1-arj3"}, lines)
	})

	t.Run("respects --include filter", func(t *testing.T) {
		path := writeMinimalKubeconfig(t, []string{"dev-use1-gkg2", "prod-usw2-ejlr", "prod-use1-arj3"})
		t.Setenv("KUBECONFIG", path)
		filterPatterns = []string{"prod"}
		t.Cleanup(func() { filterPatterns = []string{} })

		out, err := captureList(t)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(out), "\n")
		assert.Equal(t, []string{"prod-usw2-ejlr", "prod-use1-arj3"}, lines)
	})

	t.Run("respects --exclude filter", func(t *testing.T) {
		path := writeMinimalKubeconfig(t, []string{"dev-use1-gkg2", "prod-usw2-ejlr", "prod-use1-arj3"})
		t.Setenv("KUBECONFIG", path)
		excludePatterns = []string{"prod"}
		t.Cleanup(func() { excludePatterns = []string{} })

		out, err := captureList(t)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(out), "\n")
		assert.Equal(t, []string{"dev-use1-gkg2"}, lines)
	})

	t.Run("returns error when no contexts match filter", func(t *testing.T) {
		path := writeMinimalKubeconfig(t, []string{"dev-use1-gkg2"})
		t.Setenv("KUBECONFIG", path)
		filterPatterns = []string{"prod"}
		t.Cleanup(func() { filterPatterns = []string{} })

		_, err := captureList(t)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no contexts match filter patterns")
	})
}
