package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/syseng-agent/pkg/types"
)

type OpenAIRequest struct {
	Model     string     `json:"model"`
	Messages  []Message  `json:"messages"`
	Tools     []Tool     `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type AnthropicRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type AnthropicResponse struct {
	Content []Content `json:"content"`
	Error   *APIError `json:"error,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func ConvertMCPToolsToOpenAI(mcpTools []map[string]interface{}) []Tool {
	var tools []Tool
	
	for _, mcpTool := range mcpTools {
		tool := Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        getString(mcpTool, "name"),
				Description: getString(mcpTool, "description"),
				Parameters:  getMap(mcpTool, "inputSchema"),
			},
		}
		tools = append(tools, tool)
	}
	
	return tools
}

func CallOpenAI(provider *types.LLMProvider, message string) (string, error) {
	if provider.APIKey == "" {
		return "", fmt.Errorf("API key is required for OpenAI")
	}

	endpoint := "https://api.openai.com/v1/chat/completions"
	if provider.Endpoint != "" {
		endpoint = provider.Endpoint
	}

	reqBody := OpenAIRequest{
		Model: provider.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// CallOpenAIWithTools는 OpenAI Function Calling과 MCP 도구를 통합한 핵심 함수입니다
// 
// 이 함수는 사용자와 OpenAI LLM 간의 대화를 조율하며, LLM이 외부 도구(MCP 도구)를
// 반복적으로 호출하여 정보를 수집하고 작업을 수행할 수 있게 합니다.
//
// 대화 흐름:
// 1. 사용자가 질문을 합니다
// 2. LLM이 질문을 분석하고 어떤 도구를 호출할지 결정합니다
// 3. 도구가 실행되고 결과가 LLM에게 반환됩니다
// 4. LLM이 도구 결과를 처리하고 추가 도구를 호출하거나 최종 답변을 제공합니다
// 5. LLM이 완전한 응답을 제공할 때까지 이 과정이 반복됩니다
//
// 매개변수:
// - provider: LLM 설정 정보 (API 키, 모델, 엔드포인트)
// - message: 사용자의 초기 메시지/질문
// - tools: OpenAI Function 형식으로 변환된 사용 가능한 MCP 도구들
// - toolCaller: 실제 MCP 도구 호출을 실행하는 콜백 함수
//
// 컨텍스트 관리:
// - 세션 전체에 걸쳐 대화 기록(messages 배열)을 유지합니다
// - 각 도구 호출과 결과가 대화 컨텍스트의 일부가 됩니다
// - LLM은 의사결정 시 이전 도구 결과를 참조할 수 있습니다
//
// 반복 로직:
// - 복잡한 다중 도구 워크플로우를 위해 최대 10회 반복 허용
// - 각 반복: LLM 응답 → 도구 호출 → 도구 결과 → 다음 LLM 응답
// - LLM이 최종 답변 제공 시 (더 이상 도구 호출 없음) 중단
//
// OpenAI Function Calling 프로토콜:
// - LLM이 사용 가능한 도구 설명을 바탕으로 호출할 함수를 결정
// - 도구 사용 시 응답에 구조화된 tool_calls 반환
// - 우리가 도구를 실행하고 결과를 "tool" 역할 메시지로 반환
// - LLM이 결과를 처리하고 대화를 계속 진행
func CallOpenAIWithTools(provider *types.LLMProvider, message string, tools []Tool, toolCaller func(name string, args map[string]interface{}) (interface{}, error)) (string, error) {
	// 💡 API 키 검증: OpenAI API 호출을 위해서는 반드시 API 키가 필요함
	if provider.APIKey == "" {
		// 📝 조기 반환(early return): 필수 조건이 충족되지 않으면 즉시 에러 반환
		return "", fmt.Errorf("API key is required for OpenAI")
	}

	// 🌐 API 엔드포인트 설정
	// 기본값: OpenAI 공식 채팅 완성 API 엔드포인트
	endpoint := "https://api.openai.com/v1/chat/completions"
	// 🔧 사용자 정의 엔드포인트가 있으면 그것을 사용 (로컬 모델, 프록시 등)
	if provider.Endpoint != "" {
		endpoint = provider.Endpoint
	}

	// 🎯 시스템 프롬프트 정의
	// 이것은 LLM에게 "너는 누구이고 어떻게 행동해야 하는가"를 알려주는 지침서
	// 프롬프트 엔지니어링: LLM의 행동을 원하는 방향으로 유도하는 핵심 기법
	systemPrompt := `You are a system engineer AI assistant with access to powerful desktop tools. 

IMPORTANT GUIDELINES:
1. For complex questions, break them down into multiple steps and use multiple tools
2. When asked about system information (OS, uptime, processes, etc.), use multiple commands to get comprehensive data
3. Always combine results from multiple tools to provide complete answers
4. Use tools proactively - don't hesitate to gather more information if needed
5. NEVER give up after a single failure - always try alternative approaches

ERROR HANDLING STRATEGY:
• When a tool call fails, ALWAYS try alternative approaches before giving up
• If a path doesn't exist, try common alternative locations
• If a command fails, use different commands or tools to achieve the same goal
• Provide partial results rather than complete failure
• Be creative and resourceful in finding solutions

COMMON FALLBACK PATHS:
• Downloads: Try in order: ~/Downloads, ~/다운로드, ~/Desktop, ~/Documents, /tmp, $HOME
• User home: start_process("echo $HOME") or start_process("pwd") to find current location
• System info: If one command fails, use alternatives (uname, hostnamectl, sw_vers, system_profiler)
• File search: If direct path fails, use find or locate commands

ERROR RESPONSE EXAMPLES:
• "No such file or directory" → Search in alternative locations, use find command
• "Permission denied" → Try readable locations, use sudo if appropriate, or find alternative data sources
• "Command not found" → Use alternative commands or built-in shell commands
• Empty results → Try broader search, check if the query needs adjustment

AVAILABLE TOOL CATEGORIES:
- File operations: read_file, write_file, list_directory, search_files
- System info: start_process with commands like 'uname -a', 'uptime', 'ps aux'
- Process management: list_processes, start_process, interact_with_process
- Data analysis: start_process with Python/scripts for complex analysis

SYSTEM ENGINEERING EXAMPLES:
• Server info: start_process("uname -a"), start_process("uptime"), start_process("cat /etc/os-release")
• Process analysis: list_processes, start_process("ps aux"), start_process("top -n 1")  
• Disk usage: start_process("df -h"), start_process("du -sh /*")
• Network info: start_process("netstat -tuln"), start_process("ifconfig")
• Log analysis: read_file("/var/log/system.log"), search_code for patterns
• Performance: start_process("iostat"), start_process("vmstat")
• Find downloads: start_process("find ~ -name '*.download' -mtime -7"), list_directory("~/Downloads")

Always combine multiple commands and provide comprehensive analysis. When encountering errors, be persistent and creative in finding alternative solutions.`

	// 💬 대화 컨텍스트 초기화
	// OpenAI의 ChatGPT API는 대화 기록을 메시지 배열로 관리함
	// 각 메시지는 역할(role)과 내용(content)을 가짐
	messages := []Message{
		{
			// 🤖 "system" 역할: AI의 페르소나와 행동 방식을 정의
			Role:    "system",
			Content: systemPrompt, // 위에서 정의한 시스템 지침
		},
		{
			// 👤 "user" 역할: 사용자의 질문이나 요청
			Role:    "user",
			Content: message, // 함수 매개변수로 받은 사용자 메시지
		},
	}

	// 🔄 반복 제한 설정
	// 무한 루프 방지 + 복잡한 다단계 작업 허용의 균형점
	maxIterations := 10
	
	// 🎪 메인 이벤트 루프 시작
	// 각 반복에서: API 호출 → 응답 분석 → 도구 실행 → 다음 반복
	for i := 0; i < maxIterations; i++ {
		// 📦 API 요청 페이로드 준비
		// OpenAI API가 이해할 수 있는 형식으로 데이터 구조화
		reqBody := OpenAIRequest{
			Model:    provider.Model, // 🧠 사용할 AI 모델 (gpt-4, gpt-3.5-turbo 등)
			Messages: messages,       // 💬 지금까지의 전체 대화 내역
			Tools:    tools,          // 🛠️ LLM이 사용할 수 있는 도구 목록
		}

		// 🔄 Go 구조체 → JSON 변환
		// 네트워크로 전송하기 위해 바이너리 데이터로 직렬화
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			// ❌ JSON 변환 실패 시 즉시 에러 반환
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		// 🌐 HTTP POST 요청 객체 생성
		// bytes.NewBuffer: 바이트 슬라이스를 io.Reader로 변환
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			// ❌ HTTP 요청 생성 실패 시 에러 반환
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		// 📋 HTTP 헤더 설정
		req.Header.Set("Content-Type", "application/json")                    // 📝 요청 본문이 JSON임을 명시
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)            // 🔑 인증 토큰 설정

		// ⏰ HTTP 클라이언트 생성 (30초 타임아웃)
		// 타임아웃: 요청이 너무 오래 걸리면 자동으로 취소
		client := &http.Client{Timeout: 30 * time.Second}
		
		// 🚀 실제 API 호출 실행
		resp, err := client.Do(req)
		if err != nil {
			// ❌ 네트워크 에러 또는 타임아웃 시 에러 반환
			return "", fmt.Errorf("failed to make request: %w", err)
		}

		// 📥 응답 구조체 준비
		// OpenAI API 응답을 받을 Go 구조체 변수 선언
		var openAIResp OpenAIResponse
		
		// 🔄 JSON → Go 구조체 변환
		// resp.Body: HTTP 응답 본문 (JSON 형태)
		// json.NewDecoder: 스트림 방식으로 JSON 파싱 (메모리 효율적)
		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			resp.Body.Close() // 🚮 리소스 정리 (메모리 누수 방지)
			return "", fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close() // 🚮 성공 시에도 리소스 정리 필수

		// ⚠️ API 레벨 에러 체크
		// OpenAI API가 에러를 반환했는지 확인 (인증 실패, 할당량 초과 등)
		if openAIResp.Error != nil {
			return "", fmt.Errorf("API error: %s", openAIResp.Error.Message)
		}

		// 🎯 응답 유효성 검증
		// OpenAI는 여러 개의 choice를 반환할 수 있지만, 최소 1개는 있어야 함
		if len(openAIResp.Choices) == 0 {
			return "", fmt.Errorf("no response choices returned")
		}

		// 🥇 최고 점수 응답 선택
		// OpenAI는 보통 첫 번째 choice가 가장 좋은 응답
		choice := openAIResp.Choices[0]
		
		// 📚 대화 기록 업데이트
		// LLM의 응답을 메시지 배열에 추가하여 컨텍스트 유지
		// 다음 턴에서 LLM이 이전 응답을 "기억"할 수 있게 됨
		messages = append(messages, choice.Message)

		// 🔍 함수 호출 의도 감지
		// OpenAI Function Calling: LLM이 텍스트 응답 대신 함수 호출을 선택한 경우
		// ToolCalls 배열에 호출하고 싶은 함수들의 정보가 들어있음
		if len(choice.Message.ToolCalls) > 0 {
			// 🔧 도구 실행 루프
			// LLM이 여러 도구를 동시에 호출할 수 있으므로 반복 처리
			for _, toolCall := range choice.Message.ToolCalls {
				
				// 📝 함수 인수 파싱
				// LLM이 보낸 JSON 문자열을 Go map으로 변환
				// 예: '{"path": "/home/user", "recursive": true}'
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					// ❌ JSON 파싱 실패 = LLM이 잘못된 형식으로 인수를 보냄
					return "", fmt.Errorf("failed to parse tool arguments: %w", err)
				}

				// 🚀 실제 도구 실행
				// toolCaller: 우리가 구현한 콜백 함수 (MCP 도구와 연결)
				// toolCall.Function.Name: LLM이 호출하려는 함수 이름 (예: "Desktop_Commander_list_directory")
				// args: 함수에 전달할 매개변수들
				result, err := toolCaller(toolCall.Function.Name, args)
				
				// 🛡️ 에러 처리
				// 도구 실행이 실패해도 전체 대화를 중단하지 않음
				if err != nil {
					// 📝 에러를 결과 객체로 변환하여 LLM에게 알림
					// 에러 타입에 따라 구체적인 힌트 제공
					errorMsg := err.Error()
					hints := []string{}
					
					// 경로 관련 에러에 대한 힌트
					if strings.Contains(errorMsg, "no such file or directory") || 
					   strings.Contains(errorMsg, "ENOENT") {
						hints = append(hints, 
							"Try alternative paths like ~/Downloads, ~/다운로드, ~/Desktop, or use find command",
							"Use start_process('echo $HOME') to verify home directory path",
							"Search in parent directories or common locations")
					}
					
					// 권한 관련 에러에 대한 힌트
					if strings.Contains(errorMsg, "permission denied") || 
					   strings.Contains(errorMsg, "EACCES") {
						hints = append(hints,
							"Try locations with read permissions",
							"Check accessible directories with list_directory",
							"Use alternative methods to gather information")
					}
					
					// 명령어 관련 에러에 대한 힌트
					if strings.Contains(errorMsg, "command not found") {
						hints = append(hints,
							"Use alternative commands or built-in tools",
							"Try basic shell commands instead",
							"Check available tools with different approaches")
					}
					
					result = map[string]interface{}{
						"error": errorMsg,
						"hints": hints,
						"suggestion": "Please try alternative approaches based on the hints provided",
					}
				}

				// 🔄 결과 직렬화
				// Go 객체 → JSON 문자열로 변환하여 LLM에게 전달 준비
				resultJSON, _ := json.Marshal(result)
				
				// 🎯 응답 향상 로직
				// 도구 결과에 추가 컨텍스트 제공으로 더 나은 응답 유도
				toolResponse := string(resultJSON)
				
				// 빈 결과나 예상치 못한 결과에 대한 추가 지침
				if strings.Contains(toolResponse, "[]") || strings.Contains(toolResponse, "{}") {
					toolResponse += "\n\n[System: Empty result detected. Try alternative locations or methods.]"
				}
				
				// 에러가 포함된 결과에 대한 추가 강조
				if strings.Contains(toolResponse, "error") && strings.Contains(toolResponse, "ENOENT") {
					toolResponse += "\n\n[System: Path not found. You MUST try alternative paths mentioned in the hints. Do not give up!]"
				}
				
				if len(choice.Message.ToolCalls) == 1 && i < maxIterations-2 {
					// 💡 단일 도구만 사용했고 아직 반복 여유가 있으면 추가 도구 사용 유도
					toolResponse += "\n\n[System: Consider if additional tools would provide more complete information for the user's question]"
				}
				
				// 📚 도구 결과를 대화 컨텍스트에 추가
				// OpenAI Function Calling 프로토콜에 따른 메시지 구조
				messages = append(messages, Message{
					Role:       "tool",             // 🔧 "tool" 역할: 함수 실행 결과임을 OpenAI에게 알림
					Content:    toolResponse,       // 📄 실행 결과 JSON + 추가 컨텍스트
					ToolCallID: toolCall.ID,        // 🏷️ 어떤 함수 호출에 대한 응답인지 매칭 ID
				})
			}
			
			// 🔄 대화 계속 진행
			// 도구 실행이 완료되었으므로 다음 반복으로 이동
			// LLM이 도구 결과를 보고 추가 도구 호출 또는 최종 답변 결정
			continue
		}

		// 🏁 대화 종료 조건
		// LLM이 도구 호출 없이 텍스트 응답만 보냄 = 최종 답변 완성
		// 더 이상 추가 정보나 도구가 필요하지 않다고 판단
		return choice.Message.Content, nil
	}

	// ⏰ 타임아웃 상황
	// maxIterations(10회) 반복했지만 LLM이 최종 답변을 주지 않음
	// 가능한 원인: 매우 복잡한 작업, 도구 체인이 너무 길거나 LLM 혼란
	// 안전장치: 무한 루프 방지하여 시스템 안정성 보장
	return "", fmt.Errorf("maximum iterations reached without final response")
}

func CallAnthropic(provider *types.LLMProvider, message string) (string, error) {
	if provider.APIKey == "" {
		return "", fmt.Errorf("API key is required for Anthropic")
	}

	endpoint := "https://api.anthropic.com/v1/messages"
	if provider.Endpoint != "" {
		endpoint = provider.Endpoint
	}

	reqBody := AnthropicRequest{
		Model: provider.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: message,
			},
		},
		MaxTokens: 1000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if anthropicResp.Error != nil {
		return "", fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no content returned")
	}

	return anthropicResp.Content[0].Text, nil
}

func CallLocal(provider *types.LLMProvider, message string) (string, error) {
	if provider.Endpoint == "" {
		return "", fmt.Errorf("endpoint is required for local provider")
	}

	// Use OpenAI-compatible format for local providers (like Ollama with OpenAI API)
	reqBody := OpenAIRequest{
		Model: provider.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", provider.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var localResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&localResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if localResp.Error != nil {
		return "", fmt.Errorf("API error: %s", localResp.Error.Message)
	}

	if len(localResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return localResp.Choices[0].Message.Content, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}