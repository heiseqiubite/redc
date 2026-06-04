// mod/plugin/integration_test.go
package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration_ClashConfigTemplate(t *testing.T) {
	// Simulate the full flow: plugin loaded → post-apply hook → config generated

	// Setup plugin dir
	pluginDir := t.TempDir()
	os.MkdirAll(filepath.Join(pluginDir, "hooks"), 0755)

	tmpl := `mixed-port: 64277
proxies:
{{- range .IPs}}
  - name: "{{.}}"
    type: ss
    server: {{.}}
    port: {{$.Vars.port}}
    password: "{{$.Vars.password}}"
{{- end}}
{{- setOutput "clash_node_count" (printf "%d" (len .IPs)) -}}
`
	os.WriteFile(filepath.Join(pluginDir, "hooks", "post-apply.tmpl"), []byte(tmpl), 0644)

	// Setup case dir
	caseDir := t.TempDir()
	os.WriteFile(filepath.Join(caseDir, "terraform.tfvars"), []byte("port = \"8388\"\npassword = \"secret123\"\n"), 0644)

	// Terraform output
	outputs := map[string]interface{}{
		"ecs_ip": map[string]interface{}{
			"value": []interface{}{"10.0.1.1", "10.0.1.2", "10.0.1.3"},
		},
	}
	outputJSON, _ := json.Marshal(outputs)

	hook := HookEntry{
		PluginName:   "redc-plugin-clash-config",
		PluginDir:    pluginDir,
		Type:         "template",
		TemplatePath: filepath.Join(pluginDir, "hooks", "post-apply.tmpl"),
		OutputPath:   "{{.CasePath}}/config.yaml",
		Config:       map[string]interface{}{},
	}

	hctx := &HookContext{
		CaseName:   "my-proxy",
		CasePath:   caseDir,
		OutputJSON: string(outputJSON),
	}

	results, err := executeTemplateHook(hook, hctx)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	// Verify output file
	content, err := os.ReadFile(filepath.Join(caseDir, "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "10.0.1.1") {
		t.Error("missing IP 10.0.1.1")
	}
	if !strings.Contains(s, "10.0.1.3") {
		t.Error("missing IP 10.0.1.3")
	}
	if !strings.Contains(s, `password: "secret123"`) {
		t.Error("missing password")
	}
	if !strings.Contains(s, "port: 8388") {
		t.Error("missing port")
	}

	// Verify outputs
	if results["clash_node_count"] != "3" {
		t.Errorf("expected node_count=3, got %q", results["clash_node_count"])
	}
}
