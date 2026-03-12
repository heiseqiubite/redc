package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/i18n"
	"red-cloud/mod/gologger"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// readFileContent reads a file and returns its content with newlines trimmed
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// TfInit 初始化场景
func TfInit(Path string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Info().Msgf("%s", i18n.Tf("tf_init_terraform", Path))

	// 寻找执行程序
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_exec_config_failed", err))
	}

	// Use retry logic with InitRetries constant
	err = retryOperation(ctx, te.Init, InitRetries)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_network_error", err))
	}
	return nil
}

// TfInit2 复制模版后再尝试初始化
func TfInit2(Path string) error {
	if err := TfInit(Path); err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_init_failed", err))
		// 无法初始化,删除 case 文件夹
		if removeErr := os.RemoveAll(Path); removeErr != nil {
			gologger.Error().Msgf("%s", i18n.Tf("tf_init_delete_failed", removeErr))
		}
		return err // 返回原始的初始化错误
	}
	return nil
}

// RVar 统一转换接口，方便后续替换类型
func RVar(s ...string) []string {
	return s
}
func TfPlan(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Planing terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_exec_failed", err.Error()))
	}
	o := ToPlan(opts)
	// 增加 plan 输出文件
	o = append(o, tfexec.Out(RedcPlanPath))
	err = te.Plan(ctx, o...)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_create_failed", err))
		return err
	}
	err = te.ShowPlan(ctx)
	if err != nil {
		gologger.Error().Msg(i18n.T("tf_plan_show_failed"))
	}
	return nil
}
func TfApply(Path string, opts ...string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Applying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return fmt.Errorf("%s", i18n.Tf("tf_start_failed_no_tf", err))
	}
	var o []tfexec.ApplyOption
	planFile := filepath.Join(Path, RedcPlanPath)
	if _, err := os.Stat(planFile); err == nil {
		o = append(o, tfexec.DirOrPlan(RedcPlanPath))
	} else {
		o = ToApply(opts)
	}
	
	err = te.Apply(ctx, o...)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_start_failed", err.Error()))
		// 返回更详细的错误信息，包含 Terraform 的实际错误输出
		return fmt.Errorf("%s", i18n.Tf("tf_apply_failed", err))
	}
	return nil
}

func TfStatus(Path string) (*tfjson.State, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Getting terraform status in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_query_failed_no_tf", err))
	}
	s, err := te.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_query_failed", Path, err))
	}
	return s, nil
}

func TfOutput(Path string) (map[string]tfexec.OutputMeta, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Getting terraform output in %s\n", Path)

	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_exec_config_failed", err))
	}

	outputs, err := te.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_output_failed", err))
	}
	return outputs, nil
}

// TfValidate runs terraform validate to check syntax and configuration
func TfValidate(Path string) (*tfjson.ValidateOutput, error) {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Validating terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.Tf("tf_exec_config_failed", err))
	}
	result, err := te.Validate(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform validate failed: %w", err)
	}
	return result, nil
}

func TfDestroy(Path string, opts []string) error {
	ctx, cancel := createContextWithTimeout()
	defer cancel()
	gologger.Debug().Msgf("Destroying terraform in %s\n", Path)
	te, err := NewTerraformExecutor(Path)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_destroy_failed_no_tf", err))
		return err // Add return here!
	}
	if te == nil {
		return fmt.Errorf("TerraformExecutor is nil")
	}
	err = te.Destroy(ctx, ToDestroy(opts)...)
	if err != nil {
		gologger.Error().Msgf("%s", i18n.Tf("tf_destroy_failed", Path, err))
	}
	return nil
}

func ToApply(v []string) []tfexec.ApplyOption {
	var opts []tfexec.ApplyOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}

func ToPlan(v []string) []tfexec.PlanOption {
	var opts []tfexec.PlanOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}

func ToDestroy(v []string) []tfexec.DestroyOption {
	var opts []tfexec.DestroyOption
	for _, s := range v {
		opts = append(opts, tfexec.Var(s))
	}
	return opts
}
