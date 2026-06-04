// mod/plugin/template_engine_test.go
package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteTemplateHook_Basic(t *testing.T) {
	pluginDir := t.TempDir()
	os.MkdirAll(filepath.Join(pluginDir, "hooks"), 0755)
	tmplContent := `proxies:
{{- range .IPs}}
  - server: {{.}}
    port: {{$.Vars.port}}
{{- end}}
`
	os.WriteFile(filepath.Join(pluginDir, "hooks", "post-apply.tmpl"), []byte(tmplContent), 0644)

	caseDir := t.TempDir()
	os.WriteFile(filepath.Join(caseDir, "terraform.tfvars"), []byte(`port = "8388"`), 0644)

	outputs := map[string]interface{}{
		"ecs_ip": map[string]interface{}{
			"value": []interface{}{"1.2.3.4", "5.6.7.8"},
		},
	}
	outputJSON, _ := json.Marshal(outputs)

	hook := HookEntry{
		PluginName:   "test-plugin",
		PluginDir:    pluginDir,
		Type:         "template",
		TemplatePath: filepath.Join(pluginDir, "hooks", "post-apply.tmpl"),
		OutputPath:   "{{.CasePath}}/config.yaml",
		Config:       map[string]interface{}{},
	}

	hctx := &HookContext{
		CaseName:   "test-case",
		CasePath:   caseDir,
		OutputJSON: string(outputJSON),
	}

	_, err := executeTemplateHook(hook, hctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(caseDir, "config.yaml"))
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	output := string(content)
	if !strings.Contains(output, "1.2.3.4") || !strings.Contains(output, "5.6.7.8") {
		t.Errorf("output missing IPs: %s", output)
	}
	if !strings.Contains(output, "port: 8388") {
		t.Errorf("output missing port: %s", output)
	}
}

func TestExtractIPs_Array(t *testing.T) {
	outputs := map[string]interface{}{
		"ecs_ip": map[string]interface{}{
			"value": []interface{}{"10.0.0.1", "10.0.0.2"},
		},
	}
	ips := extractIPs(outputs)
	if len(ips) != 2 || ips[0] != "10.0.0.1" {
		t.Errorf("got %v", ips)
	}
}

func TestExtractIPs_String(t *testing.T) {
	outputs := map[string]interface{}{
		"public_ip": map[string]interface{}{
			"value": "192.168.1.1",
		},
	}
	ips := extractIPs(outputs)
	if len(ips) != 1 || ips[0] != "192.168.1.1" {
		t.Errorf("got %v", ips)
	}
}

func TestExtractIPs_Empty(t *testing.T) {
	outputs := map[string]interface{}{}
	ips := extractIPs(outputs)
	if len(ips) != 0 {
		t.Errorf("expected empty, got %v", ips)
	}
}

func TestExecuteTemplateHook_SetOutput(t *testing.T) {
	pluginDir := t.TempDir()
	os.MkdirAll(filepath.Join(pluginDir, "hooks"), 0755)
	tmplContent := `{{setOutput "key1" "value1"}}{{setOutput "key2" "value2"}}done`
	os.WriteFile(filepath.Join(pluginDir, "hooks", "test.tmpl"), []byte(tmplContent), 0644)

	hook := HookEntry{
		PluginName:   "test",
		PluginDir:    pluginDir,
		Type:         "template",
		TemplatePath: filepath.Join(pluginDir, "hooks", "test.tmpl"),
		OutputPath:   "",
		Config:       map[string]interface{}{},
	}

	results, err := executeTemplateHook(hook, &HookContext{CasePath: t.TempDir()})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if results["key1"] != "value1" || results["key2"] != "value2" {
		t.Errorf("unexpected results: %v", results)
	}
}
