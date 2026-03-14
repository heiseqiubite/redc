package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/ai"
	"red-cloud/mod/mcp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// agentCancelMap stores cancel functions for active agent conversations
var agentCancelMap = struct {
	sync.Mutex
	m map[string]context.CancelFunc
}{m: make(map[string]context.CancelFunc)}

// askUserChannels stores channels for ask_user tool responses, keyed by conversationId
var askUserChannels = struct {
	sync.Mutex
	m map[string]chan string
}{m: make(map[string]chan string)}

// AIChatMessage represents a single message in the AI chat conversation
type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIChatStream handles multi-turn AI chat with streaming responses
func (a *App) AIChatStream(conversationId, mode string, messages []AIChatMessage) error {
	// Validate AI config
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}

	// Determine system prompt based on mode
	var systemPrompt string
	switch mode {
	case "generate":
		systemPrompt = ai.TemplateGenerationSystemPrompt + "\n\n" + langPrompt

	case "recommend":
		localTemplates, _ := redc.ListLocalTemplates()
		templateList := make([]string, 0, len(localTemplates))
		for _, t := range localTemplates {
			templateList = append(templateList, fmt.Sprintf("- %s: %s", t.Name, t.Description))
		}
		systemPrompt = fmt.Sprintf(ai.TemplateRecommendationSystemPrompt,
			strings.Join(templateList, "\n"),
			langPrompt)

	case "cost":
		systemPrompt = fmt.Sprintf(ai.CostOptimizationSystemPrompt, langPrompt)
		// Gather running cases info and prepend to the last user message
		casesInfo, runningCount := a.gatherRunningCasesInfo()
		if runningCount > 0 {
			userPrompt := fmt.Sprintf(ai.CostOptimizationUserPrompt, runningCount, casesInfo)
			// Prepend context to the last user message
			if len(messages) > 0 {
				lastIdx := len(messages) - 1
				messages[lastIdx].Content = userPrompt + "\n\n用户额外说明：" + messages[lastIdx].Content
			}
		}

	case "errorAnalysis":
		// Try to read template content for context
		templateContext := ""
		if len(messages) > 0 {
			// Extract template name from the first user message
			firstMsg := messages[0].Content
			if idx := strings.Index(firstMsg, "模板: "); idx >= 0 {
				end := strings.Index(firstMsg[idx+len("模板: "):], "\n")
				var tmplName string
				if end >= 0 {
					tmplName = firstMsg[idx+len("模板: ") : idx+len("模板: ")+end]
				} else {
					tmplName = firstMsg[idx+len("模板: "):]
				}
				tmplName = strings.TrimSpace(tmplName)
				if tmplName != "" {
					templateContext = a.gatherTemplateContext(tmplName)
				}
			}
		}
		systemPrompt = fmt.Sprintf(ai.ErrorAnalysisChatSystemPrompt, templateContext, langPrompt)

	case "free":
		systemPrompt = fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)

	default:
		systemPrompt = fmt.Sprintf(ai.FreeChatSystemPrompt, langPrompt)
	}

	// Build ai.Message slice: system prompt + user-provided history
	aiMessages := make([]ai.Message, 0, len(messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = client.ChatStream(ctx, aiMessages, func(chunk string) error {
		a.emitEvent( "ai-chat-chunk", map[string]string{
			"conversationId": conversationId,
			"chunk":          chunk,
		})
		return nil
	})

	if err != nil {
		a.emitEvent( "ai-chat-complete", map[string]interface{}{
			"conversationId": conversationId,
			"success":        false,
		})
		return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
	}

	a.emitEvent( "ai-chat-complete", map[string]interface{}{
		"conversationId": conversationId,
		"success":        true,
	})
	return nil
}

// AgentChatStream runs the agentic loop: AI + MCP tool calling + streaming final answer
func (a *App) AgentChatStream(conversationId string, messages []AIChatMessage) error {
	return a.runAgentLoop(conversationId, messages, ai.AgentSystemPrompt, 50, 10*time.Minute)
}

// DeployAgentChatStream runs the deploy agent loop with specialized system prompt
func (a *App) DeployAgentChatStream(conversationId string, messages []AIChatMessage) error {
	return a.runAgentLoop(conversationId, messages, ai.DeployAgentSystemPrompt, 50, 15*time.Minute)
}

// StopAgentStream cancels a running agent conversation
func (a *App) StopAgentStream(conversationId string) {
	agentCancelMap.Lock()
	if cancel, ok := agentCancelMap.m[conversationId]; ok {
		cancel()
		delete(agentCancelMap.m, conversationId)
	}
	agentCancelMap.Unlock()
}

// SubmitAskUserResponse sends user's answer back to a waiting ask_user tool call
func (a *App) SubmitAskUserResponse(conversationId string, answer string) {
	askUserChannels.Lock()
	ch, ok := askUserChannels.m[conversationId]
	askUserChannels.Unlock()
	if ok {
		select {
		case ch <- answer:
		default:
		}
	}
}

// ExportChatLog saves chat log content to a user-selected file
func (a *App) ExportChatLog(content string) error {
	filename := fmt.Sprintf("redc-chat-%s.md", time.Now().Format("20060102-150405"))
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出对话日志",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return nil
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// runAgentLoop is the shared agentic loop used by AgentChatStream and DeployAgentChatStream
func (a *App) runAgentLoop(conversationId string, messages []AIChatMessage, promptTemplate string, defaultMaxRounds int, timeout time.Duration) error {
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}
	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	// Use user-configured max rounds if set, otherwise use default (cap at 200)
	maxRounds := defaultMaxRounds
	if aiConfig.MaxToolRounds > 0 {
		maxRounds = aiConfig.MaxToolRounds
	}
	if maxRounds > 200 {
		maxRounds = 200
	}

	a.mu.Lock()
	project := a.project
	a.mu.Unlock()
	if project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}
	systemPrompt := fmt.Sprintf(promptTemplate, langPrompt)

	// Build tool definitions from MCP server
	mcpServer := mcp.NewMCPServer(project, a)
	mcpTools := mcpServer.GetTools()
	enableAskUser := aiConfig.EnableAskUser == nil || *aiConfig.EnableAskUser // default true
	toolDefs := make([]ai.ToolDefinition, 0, len(mcpTools))
	for _, t := range mcpTools {
		if t.Name == "ask_user" && !enableAskUser {
			continue
		}
		params := map[string]interface{}{
			"type":       t.InputSchema.Type,
			"properties": t.InputSchema.Properties,
		}
		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}
		toolDefs = append(toolDefs, ai.ToolDefinition{
			Type: "function",
			Function: ai.ToolFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	// Build initial message list: system + history
	aiMessages := make([]ai.Message, 0, len(messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Register cancel function so frontend can stop this conversation
	agentCancelMap.Lock()
	agentCancelMap.m[conversationId] = cancel
	agentCancelMap.Unlock()
	defer func() {
		agentCancelMap.Lock()
		delete(agentCancelMap.m, conversationId)
		agentCancelMap.Unlock()
	}()

	// Agentic loop
	for round := 0; round < maxRounds; round++ {
		// Compress old tool results to prevent context window bloat
		// Keep system prompt + user messages + recent tool interactions intact,
		// but shorten tool results from rounds older than the last 2
		if round > 2 {
			compressOldToolResults(aiMessages)
		}

		// Check if cancelled before each round
		if ctx.Err() != nil {
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          "\n\n⏹️ 操作已被用户停止。",
			})
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        true,
			})
			return nil
		}

		// Use streaming tool call so users can see AI thinking in real-time
		resp, err := client.ChatWithToolsStream(ctx, aiMessages, toolDefs, func(chunk string) error {
			a.emitEvent("ai-chat-chunk", map[string]string{
				"conversationId": conversationId,
				"chunk":          chunk,
			})
			return nil
		})
		if err != nil {
			if ctx.Err() != nil {
				a.emitEvent("ai-chat-chunk", map[string]string{
					"conversationId": conversationId,
					"chunk":          "\n\n⏹️ 操作已被用户停止。",
				})
				a.emitEvent("ai-chat-complete", map[string]interface{}{
					"conversationId": conversationId,
					"success":        true,
				})
				return nil
			}
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        false,
			})
			return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
		}

		// No tool calls → final answer (already streamed via callback)
		if len(resp.ToolCalls) == 0 {
			aiMessages = append(aiMessages, ai.Message{Role: "assistant", Content: resp.Content})
			a.emitEvent("ai-chat-complete", map[string]interface{}{
				"conversationId": conversationId,
				"success":        true,
			})
			return nil
		}

		// Append assistant message with tool_calls to history
		aiMessages = append(aiMessages, ai.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			var args map[string]interface{}
			var jsonParseErr error
			if tc.Function.Arguments != "" {
				if jsonErr := json.Unmarshal([]byte(tc.Function.Arguments), &args); jsonErr != nil {
					// Streaming may produce incomplete JSON for no-arg tools; treat as empty
					trimmed := strings.TrimSpace(tc.Function.Arguments)
					if trimmed == "{" || trimmed == "" {
						args = map[string]interface{}{}
					} else {
						jsonParseErr = jsonErr
						args = map[string]interface{}{}
					}
				}
			}

			a.emitEvent("ai-agent-tool-call", map[string]interface{}{
				"conversationId": conversationId,
				"toolCallId":     tc.ID,
				"toolName":       tc.Function.Name,
				"toolArgs":       args,
			})

			var resultContent string
			var success bool

			if jsonParseErr != nil {
				// Report JSON parse failure as tool result so AI knows the root cause
				resultContent = fmt.Sprintf("工具参数 JSON 解析失败: %v\n原始参数: %s", jsonParseErr, tc.Function.Arguments)
				success = false
			} else if tc.Function.Name == "ask_user" {
				// Special handling: pause loop, emit event, wait for user response
				question, _ := args["question"].(string)
				var choices []string
				if rawChoices, ok := args["choices"].([]interface{}); ok {
					for _, c := range rawChoices {
						if s, ok := c.(string); ok {
							choices = append(choices, s)
						}
					}
				}
				allowFreeform := true
				if af, ok := args["allow_freeform"].(bool); ok {
					allowFreeform = af
				}

				// Create response channel
				ch := make(chan string, 1)
				askUserChannels.Lock()
				askUserChannels.m[conversationId] = ch
				askUserChannels.Unlock()
				defer func() {
					askUserChannels.Lock()
					delete(askUserChannels.m, conversationId)
					askUserChannels.Unlock()
				}()

				// Emit ask_user event to frontend
				a.emitEvent("ai-agent-ask-user", map[string]interface{}{
					"conversationId": conversationId,
					"toolCallId":     tc.ID,
					"question":       question,
					"choices":        choices,
					"allowFreeform":  allowFreeform,
				})

				// Wait for user response or cancellation
				select {
				case answer := <-ch:
					resultContent = answer
					success = true
				case <-ctx.Done():
					resultContent = "用户未回答，操作已取消"
					success = false
				}
			} else {
				result, execErr := mcpServer.ExecuteTool(tc.Function.Name, args)
				success = execErr == nil
				if execErr != nil {
					resultContent = fmt.Sprintf("工具执行失败: %v", execErr)
				} else if len(result.Content) > 0 {
					var parts []string
					for _, item := range result.Content {
						parts = append(parts, item.Text)
					}
					resultContent = strings.Join(parts, "\n")
				}
			}

			// Truncate large tool results to prevent context window overflow
			const maxToolResultLen = 8000
			if len(resultContent) > maxToolResultLen {
				resultContent = resultContent[:maxToolResultLen] + "\n\n... (output truncated, total " + fmt.Sprintf("%d", len(resultContent)) + " bytes)"
			}

			a.emitEvent("ai-agent-tool-result", map[string]interface{}{
				"conversationId": conversationId,
				"toolCallId":     tc.ID,
				"toolName":       tc.Function.Name,
				"success":        success,
				"content":        resultContent,
			})

			aiMessages = append(aiMessages, ai.Message{
				Role:       "tool",
				Content:    resultContent,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	// Exceeded max rounds
	a.emitEvent( "ai-chat-chunk", map[string]string{
		"conversationId": conversationId,
		"chunk":          fmt.Sprintf("\n\n⚠️ 已达到最大工具调用轮次（%d轮），操作结束。", maxRounds),
	})
	a.emitEvent( "ai-chat-complete", map[string]interface{}{
		"conversationId": conversationId,
		"success":        true,
	})
	return nil
}

// compressOldToolResults shortens tool result messages beyond the most recent round
// to prevent context window bloat over many rounds.
// It keeps the last 4 tool-role messages at full length and compresses older ones.
func compressOldToolResults(messages []ai.Message) {
	const maxCompressedLen = 200
	const keepRecentToolMessages = 4

	// Count total tool messages
	toolCount := 0
	for i := range messages {
		if messages[i].Role == "tool" {
			toolCount++
		}
	}
	if toolCount <= keepRecentToolMessages {
		return
	}

	// Compress older tool messages (keep last keepRecentToolMessages intact)
	skipCount := toolCount - keepRecentToolMessages
	compressed := 0
	for i := 0; i < len(messages); i++ {
		if messages[i].Role == "tool" && compressed < skipCount {
			content := messages[i].Content
			if len(content) > maxCompressedLen {
				messages[i].Content = content[:maxCompressedLen] + "... (compressed)"
			}
			compressed++
		}
	}
}

func (a *App) gatherRunningCasesInfo() (string, int) {
	a.mu.Lock()
	project := a.project
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	a.mu.Unlock()

	if project == nil || pricingService == nil || costCalculator == nil {
		return "", 0
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return "", 0
	}

	var caseInfoList []string
	runningCount := 0

	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		runningCount++

		if c.Path == "" {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 场景路径为空
  - 建议: 请检查场景配置`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		state, err := redc.TfStatus(c.Path)
		if err != nil {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态获取失败 (%v)
  - 建议: 请检查 Terraform 是否正确安装，场景是否已完成部署`, c.Name, c.Module, err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if state == nil || state.Values == nil {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态数据为空
  - 建议: 该场景可能尚未创建资源`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		resources := extractResourcesFromState(state)
		if resources == nil || len(resources.Resources) == 0 {
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 未找到资源信息
  - 建议: 该场景可能尚未创建资源，或资源已被销毁`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		estimate, err := costCalculator.CalculateCost(resources, pricingService)
		if err != nil {
			var resourceList []string
			for _, r := range resources.Resources {
				resourceList = append(resourceList, fmt.Sprintf("  - %s (%s)", r.Name, r.Type))
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 资源数量: %d
  - 资源列表:
%s
  - 说明: 成本计算失败 (%v)
  - 建议: 请检查定价数据是否可用`, c.Name, c.Module, len(resources.Resources), strings.Join(resourceList, "\n"), err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		var resourceDetails []string
		for _, rb := range estimate.Breakdown {
			if rb.TotalMonthly > 0 {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): ¥%.2f/月",
					rb.ResourceName, rb.ResourceType, rb.TotalMonthly))
			} else if !rb.Available {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): 定价不可用",
					rb.ResourceName, rb.ResourceType))
			}
		}

		provider := "未知"
		if len(estimate.Breakdown) > 0 {
			provider = estimate.Breakdown[0].Provider
		}

		caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 云服务商: %s
  - 月度成本: ¥%.2f
  - 资源数量: %d
  - 资源详情:
%s`, c.Name, c.Module, provider, estimate.TotalMonthlyCost, len(estimate.Breakdown), strings.Join(resourceDetails, "\n"))

		caseInfoList = append(caseInfoList, caseInfo)
	}

	return strings.Join(caseInfoList, "\n\n"), runningCount
}

// gatherTemplateContext reads template files and returns context for error analysis
func (a *App) gatherTemplateContext(templateName string) string {
	tmplPath, err := redc.GetTemplatePath(templateName)
	if err != nil {
		return ""
	}
	entries, err := os.ReadDir(tmplPath)
	if err != nil {
		return ""
	}

	var parts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "case.json" || strings.HasSuffix(name, ".tf") || name == "terraform.tfvars" {
			data, err := os.ReadFile(filepath.Join(tmplPath, name))
			if err != nil {
				continue
			}
			content := string(data)
			// Truncate very large files
			if len(content) > 3000 {
				content = content[:3000] + "\n... (truncated)"
			}
			parts = append(parts, fmt.Sprintf("### %s\n```\n%s\n```", name, content))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "## 当前模板文件内容（供参考）\n\n" + strings.Join(parts, "\n\n")
}
