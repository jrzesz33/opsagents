package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"opsagents/internal/config"
	"opsagents/pkg/agent"
	"opsagents/pkg/deploy"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "opsagents",
		Short: "OpsAgents - Claude AI Agent for DevOps Automation",
		Long:  `An intelligent Claude AI agent that automates deploying pre-built applications to AWS ECS Fargate with natural language commands`,
	}

	var agentCmd = &cobra.Command{
		Use:   "agent",
		Short: "Start the Claude AI agent",
		Long:  `Start an interactive session with the Claude AI agent to deploy applications`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runAgent(); err != nil {
				fmt.Printf("Agent failed: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy to AWS ECS Fargate (direct mode)",
		Long:  `Deploy Docker containers to AWS ECS Fargate`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting deployment...")
			if err := runDeploy(); err != nil {
				fmt.Printf("Deployment failed: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Generate default configuration",
		Long:  `Generate a default config.yaml file`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := config.CreateDefaultConfig(); err != nil {
				fmt.Printf("Failed to create config: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runAgent() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	claudeAgent, err := agent.NewClaudeAgent(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Claude agent: %w", err)
	}

	fmt.Println("🤖 Claude OpsAgent - Your AI DevOps Assistant")
	fmt.Println("Type 'exit' or 'quit' to stop the agent")
	fmt.Println("Available commands:")
	fmt.Println("  - 'deploy to production' - Deploy pre-built containers to AWS Lightsail")
	fmt.Println("  - 'check deployment status' - Get current deployment status")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		fmt.Print("🤖 Claude: ")
		response, err := claudeAgent.SendMessage(context.Background(), input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(response)
		fmt.Println()
	}

	return nil
}


func runDeploy() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize ECS deployer
	deployer, err := deploy.NewECSDeployer()
	if err != nil {
		return fmt.Errorf("failed to initialize ECS deployer: %w", err)
	}

	// Create ECS configuration
	ecsConfig := deploy.ECSConfig{
		ClusterName:        cfg.AWS.ECS.ClusterName,
		ServiceName:        cfg.AWS.ECS.ServiceName,
		TaskDefinitionName: cfg.AWS.ECS.TaskDefinitionName,
		VpcId:              cfg.AWS.ECS.VpcId,
		SubnetIds:          cfg.AWS.ECS.SubnetIds,
		SecurityGroupIds:   cfg.AWS.ECS.SecurityGroupIds,
		LoadBalancerName:   cfg.AWS.ECS.LoadBalancerName,
		WebAppImage:        cfg.Images.AppImage,
		DatabaseImage:      cfg.Images.Neo4jImage,
		WebAppPort:         cfg.AWS.ECS.WebAppPort,
		DatabasePort:       cfg.AWS.ECS.DatabasePort,
		WebAppMemory:       cfg.AWS.ECS.WebAppMemory,
		WebAppCPU:          cfg.AWS.ECS.WebAppCPU,
		DatabaseMemory:     cfg.AWS.ECS.DatabaseMemory,
		DatabaseCPU:        cfg.AWS.ECS.DatabaseCPU,
		Environment:        cfg.AWS.ECS.Environment,
	}

	// Create ECS cluster
	if err := deployer.CreateCluster(ecsConfig.ClusterName); err != nil {
		fmt.Printf("ECS cluster might already exist: %v\n", err)
	}

	// Create task definition
	if err := deployer.CreateTaskDefinition(ecsConfig); err != nil {
		return fmt.Errorf("failed to create task definition: %w", err)
	}

	// Create ECS service
	if err := deployer.CreateService(ecsConfig); err != nil {
		return fmt.Errorf("failed to create ECS service: %w", err)
	}

	// Wait for service to be stable
	if err := deployer.WaitForServiceStable(ecsConfig.ClusterName, ecsConfig.ServiceName); err != nil {
		return fmt.Errorf("failed waiting for service to be stable: %w", err)
	}

	fmt.Println("Deployment completed successfully!")
	return nil
}
