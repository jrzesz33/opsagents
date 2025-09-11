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
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"os"
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
	// Load AWS config with explicit environment variable credentials
	var awsConfig aws.Config
	var err error
	
	// Check if we have environment variables for AWS credentials
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	
	if region == "" {
		region = cfg.Claude.Region
	}
	
	if accessKey != "" && secretKey != "" {
		// Use static credentials from environment variables
		awsConfig, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
	} else {
		// Fall back to default credential chain (excluding IMDS)
		awsConfig, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(region),
			awsconfig.WithEC2IMDSClientEnableState(imds.ClientDisabled),
		)
	}
	
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
		{
			Name:        "cleanup_resources",
			Description: "Clean up all AWS ECS resources including services, clusters, load balancers, and log groups",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Set to true to confirm resource deletion",
					},
					"service_name": map[string]interface{}{
						"type":        "string",
						"description": "Optional service name to clean up specific resources",
					},
				},
				Required: []string{"confirm"},
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
	case "cleanup_resources":
		return a.executeCleanupTool(toolUse)
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

	// Initialize ECS deployer
	deployer, err := deploy.NewECSDeployer()
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to initialize ECS deployer: %v", err),
		}, nil
	}

	// Get service name from input or use default
	serviceName := a.config.AWS.ECS.ServiceName
	if name, ok := toolUse.Input["service_name"].(string); ok && name != "" {
		serviceName = name
	}

	// Create ECS configuration
	ecsConfig := deploy.ECSConfig{
		ClusterName:        a.config.AWS.ECS.ClusterName,
		ServiceName:        serviceName,
		TaskDefinitionName: a.config.AWS.ECS.TaskDefinitionName,
		VpcId:              a.config.AWS.ECS.VpcId,
		SubnetIds:          a.config.AWS.ECS.SubnetIds,
		SecurityGroupIds:   a.config.AWS.ECS.SecurityGroupIds,
		LoadBalancerName:   a.config.AWS.ECS.LoadBalancerName,
		WebAppImage:        a.config.Images.AppImage,
		DatabaseImage:      a.config.Images.Neo4jImage,
		WebAppPort:         a.config.AWS.ECS.WebAppPort,
		DatabasePort:       a.config.AWS.ECS.DatabasePort,
		DatabaseHTTPPort:   a.config.AWS.ECS.DatabaseHTTPPort,
		WebAppMemory:       a.config.AWS.ECS.WebAppMemory,
		WebAppCPU:          a.config.AWS.ECS.WebAppCPU,
		DatabaseMemory:     a.config.AWS.ECS.DatabaseMemory,
		DatabaseCPU:        a.config.AWS.ECS.DatabaseCPU,
		Environment:        a.config.AWS.ECS.Environment,
		CreateSecrets:      a.config.AWS.ECS.CreateSecrets,
		CreateEFS:          a.config.AWS.ECS.CreateEFS,
		EFSVolumeId:        a.config.AWS.ECS.EFSVolumeId,
		Mode:               a.config.AWS.ECS.Mode,
	}

	// Create ECS cluster
	if err := deployer.CreateCluster(ecsConfig.ClusterName); err != nil {
		log.Printf("ECS cluster might already exist: %v", err)
	}

	// Use advanced deployment if advanced features are enabled
	if ecsConfig.CreateSecrets || ecsConfig.CreateEFS {
		if err := deployer.DeployAdvanced(ecsConfig); err != nil {
			return &ToolResult{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Failed to deploy with advanced features: %v", err),
			}, nil
		}
	} else {
		// Basic deployment
		if err := deployer.CreateTaskDefinition(ecsConfig); err != nil {
			return &ToolResult{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Failed to create task definition: %v", err),
			}, nil
		}
	}

	// Create ECS service (for both advanced and basic deployments)
	if err := deployer.CreateService(ecsConfig); err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to create ECS service: %v", err),
		}, nil
	}

	// Wait for service to be stable if requested
	waitForReady := true
	if wait, ok := toolUse.Input["wait_for_ready"].(bool); ok {
		waitForReady = wait
	}

	if waitForReady {
		if err := deployer.WaitForServiceStable(ecsConfig.ClusterName, serviceName); err != nil {
			return &ToolResult{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Failed waiting for service to be stable: %v", err),
			}, nil
		}
	}

	return &ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUse.ID,
		Content:   fmt.Sprintf("ECS deployment to service '%s' completed successfully!", serviceName),
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

	// Initialize ECS deployer
	deployer, err := deploy.NewECSDeployer()
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to initialize ECS deployer: %v", err),
		}, nil
	}

	// Get service status
	status, err := deployer.GetServiceStatus(a.config.AWS.ECS.ClusterName, serviceName)
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to get service status: %v", err),
		}, nil
	}

	return &ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUse.ID,
		Content:   status,
	}, nil
}

func (a *ClaudeAgent) executeCleanupTool(toolUse ToolUse) (*ToolResult, error) {
	log.Println("Executing cleanup_resources tool")

	// Check confirmation
	confirm, ok := toolUse.Input["confirm"].(bool)
	if !ok || !confirm {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   "Cleanup cancelled. The 'confirm' parameter must be set to true to proceed with resource deletion.",
		}, nil
	}

	// Initialize ECS deployer
	deployer, err := deploy.NewECSDeployer()
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Failed to initialize ECS deployer: %v", err),
		}, nil
	}

	// Get service name from input or use default
	serviceName := a.config.AWS.ECS.ServiceName
	if name, ok := toolUse.Input["service_name"].(string); ok && name != "" {
		serviceName = name
	}

	// Create ECS configuration
	ecsConfig := deploy.ECSConfig{
		ClusterName:        a.config.AWS.ECS.ClusterName,
		ServiceName:        serviceName,
		TaskDefinitionName: a.config.AWS.ECS.TaskDefinitionName,
		VpcId:              a.config.AWS.ECS.VpcId,
		SubnetIds:          a.config.AWS.ECS.SubnetIds,
		SecurityGroupIds:   a.config.AWS.ECS.SecurityGroupIds,
		LoadBalancerName:   a.config.AWS.ECS.LoadBalancerName,
		WebAppImage:        a.config.Images.AppImage,
		DatabaseImage:      a.config.Images.Neo4jImage,
		WebAppPort:         a.config.AWS.ECS.WebAppPort,
		DatabasePort:       a.config.AWS.ECS.DatabasePort,
		DatabaseHTTPPort:   a.config.AWS.ECS.DatabaseHTTPPort,
		WebAppMemory:       a.config.AWS.ECS.WebAppMemory,
		WebAppCPU:          a.config.AWS.ECS.WebAppCPU,
		DatabaseMemory:     a.config.AWS.ECS.DatabaseMemory,
		DatabaseCPU:        a.config.AWS.ECS.DatabaseCPU,
		Environment:        a.config.AWS.ECS.Environment,
		CreateSecrets:      a.config.AWS.ECS.CreateSecrets,
		CreateEFS:          a.config.AWS.ECS.CreateEFS,
		EFSVolumeId:        a.config.AWS.ECS.EFSVolumeId,
		Mode:               a.config.AWS.ECS.Mode,
	}

	// Execute cleanup
	err = deployer.Cleanup(ecsConfig)
	if err != nil {
		return &ToolResult{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   fmt.Sprintf("Cleanup failed: %v", err),
		}, nil
	}

	return &ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUse.ID,
		Content:   fmt.Sprintf("✅ Successfully cleaned up all AWS ECS resources for service '%s'! This included:\n- ECS Service and Tasks\n- Task Definitions\n- Load Balancer and Target Groups\n- CloudWatch Log Groups\n- ECS Cluster (if empty)", serviceName),
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
