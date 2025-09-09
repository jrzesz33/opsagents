package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"opsagents/internal/config"
	"opsagents/pkg/deploy"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

type ClaudeAgent struct {
	client      *bedrockruntime.Client
	config      *config.Config
	modelID     string
	temperature float32
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

type ToolUse struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type ToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

func NewClaudeAgent(cfg *config.Config) (*ClaudeAgent, error) {
	// Load AWS config with environment variables
	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.Claude.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(awsConfig)

	return &ClaudeAgent{
		client:      client,
		config:      cfg,
		modelID:     cfg.Claude.ModelID,
		temperature: cfg.Claude.Temperature,
	}, nil
}

func (a *ClaudeAgent) GetTools() []Tool {
	return []Tool{
		{
			Name:        "deploy_application",
			Description: "Deploy the application containers to AWS Lightsail",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"service_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the Lightsail service to deploy to",
					},
					"wait_for_ready": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to wait for service to become ready",
					},
				},
				Required: []string{},
			},
		},
		{
			Name:        "get_deployment_status",
			Description: "Get the current status of a deployed application",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"service_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the Lightsail service to check",
					},
				},
				Required: []string{"service_name"},
			},
		},
	}
}

func (a *ClaudeAgent) ExecuteTool(toolUse ToolUse) (*ToolResult, error) {
	switch toolUse.Name {
	case "deploy_application":
		return a.executeDeployTool(toolUse)
	case "get_deployment_status":
		return a.executeStatusTool(toolUse)
	default:
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Unknown tool: %s", toolUse.Name),
		}, nil
	}
}


func (a *ClaudeAgent) executeDeployTool(toolUse ToolUse) (*ToolResult, error) {
	log.Println("Executing deploy_application tool")

	// Initialize Lightsail deployer
	deployer, err := deploy.NewLightsailDeployer()
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to initialize Lightsail deployer: %v", err),
		}, nil
	}

	// Get service name from input or use default
	serviceName := a.config.AWS.Lightsail.ServiceName
	if name, ok := toolUse.Input["service_name"].(string); ok && name != "" {
		serviceName = name
	}

	// Create container service configuration
	serviceConfig := deploy.ContainerServiceConfig{
		ServiceName:   serviceName,
		Power:         types.ContainerServicePowerName(a.config.AWS.Lightsail.Power),
		Scale:         a.config.AWS.Lightsail.Scale,
		PublicDomain:  a.config.AWS.Lightsail.PublicDomain,
		ContainerName: a.config.AWS.Lightsail.ContainerName,
		ImageName:     a.config.Images.AppImage,
		Ports: map[string]int32{
			"8080": 8080,
		},
		Environment: a.config.AWS.Lightsail.Environment,
	}

	// Create container service
	if err := deployer.CreateContainerService(serviceConfig); err != nil {
		log.Printf("Container service might already exist: %v", err)
	}

	// Deploy container
	if err := deployer.DeployContainer(serviceName, serviceConfig); err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to deploy container: %v", err),
		}, nil
	}

	// Wait for service to be ready if requested
	waitForReady := true
	if wait, ok := toolUse.Input["wait_for_ready"].(bool); ok {
		waitForReady = wait
	}

	if waitForReady {
		if err := deployer.WaitForServiceReady(serviceName); err != nil {
			return &ToolResult{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Failed waiting for service to be ready: %v", err),
			}, nil
		}
	}

	return &ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUse.ID,
		Content:   fmt.Sprintf("Deployment to service '%s' completed successfully!", serviceName),
	}, nil
}

func (a *ClaudeAgent) executeStatusTool(toolUse ToolUse) (*ToolResult, error) {
	log.Println("Executing get_deployment_status tool")

	serviceName, ok := toolUse.Input["service_name"].(string)
	if !ok {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   "service_name parameter is required",
		}, nil
	}

	// Initialize Lightsail deployer
	deployer, err := deploy.NewLightsailDeployer()
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to initialize Lightsail deployer: %v", err),
		}, nil
	}

	// Get service state
	service, err := deployer.GetContainerServiceState(serviceName)
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to get service state: %v", err),
		}, nil
	}

	status := fmt.Sprintf("Service: %s\nState: %s", serviceName, service.State)
	if service.Url != nil {
		status += fmt.Sprintf("\nURL: %s", *service.Url)
	}
	if service.CurrentDeployment != nil {
		status += fmt.Sprintf("\nDeployment State: %s", service.CurrentDeployment.State)
	}

	return &ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUse.ID,
		Content:   status,
	}, nil
}

func (a *ClaudeAgent) SendMessage(ctx context.Context, message string) (string, error) {
	tools := a.GetTools()

	requestBody := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        4096,
		"temperature":       a.temperature,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": message,
			},
		},
		"tools": tools,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(a.modelID),
		ContentType: aws.String("application/json"),
		Body:        requestJSON,
	}

	result, err := a.client.InvokeModel(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to invoke model: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Process the response and handle tool calls
	return a.processResponse(ctx, response)
}

func (a *ClaudeAgent) processResponse(ctx context.Context, response map[string]interface{}) (string, error) {
	content, ok := response["content"].([]interface{})
	if !ok {
		return "No content in response", nil
	}

	var textResponse string
	var toolCalls []ToolUse

	// Parse response content
	for _, item := range content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		switch itemMap["type"] {
		case "text":
			if text, ok := itemMap["text"].(string); ok {
				textResponse += text
			}
		case "tool_use":
			toolUse := ToolUse{
				Type:  "tool_use",
				ID:    itemMap["id"].(string),
				Name:  itemMap["name"].(string),
				Input: itemMap["input"].(map[string]interface{}),
			}
			toolCalls = append(toolCalls, toolUse)
		}
	}

	// Execute tool calls if any
	if len(toolCalls) > 0 {
		var results []string
		for _, toolCall := range toolCalls {
			result, err := a.ExecuteTool(toolCall)
			if err != nil {
				results = append(results, fmt.Sprintf("Tool %s failed: %v", toolCall.Name, err))
			} else {
				results = append(results, result.Content)
			}
		}

		if textResponse != "" {
			textResponse += "\n\n"
		}
		textResponse += "Tool Results:\n" + fmt.Sprintf("%v", results)
	}

	return textResponse, nil
}
