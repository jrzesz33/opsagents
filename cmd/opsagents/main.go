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

	var cleanupCmd = &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up AWS ECS resources",
		Long:  `Remove all AWS ECS resources including services, clusters, load balancers, and log groups`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting cleanup of AWS ECS resources...")
			if err := runCleanup(); err != nil {
				fmt.Printf("Cleanup failed: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(cleanupCmd)

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
	fmt.Println("  - 'deploy to production' - Deploy pre-built containers to AWS ECS")
	fmt.Println("  - 'check deployment status' - Get current deployment status")
	fmt.Println("  - 'cleanup resources' - Remove all AWS ECS resources")
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
		DatabaseHTTPPort:   cfg.AWS.ECS.DatabaseHTTPPort,
		WebAppMemory:       cfg.AWS.ECS.WebAppMemory,
		WebAppCPU:          cfg.AWS.ECS.WebAppCPU,
		DatabaseMemory:     cfg.AWS.ECS.DatabaseMemory,
		DatabaseCPU:        cfg.AWS.ECS.DatabaseCPU,
		Environment:        cfg.AWS.ECS.Environment,
		CreateSecrets:      cfg.AWS.ECS.CreateSecrets,
		CreateEFS:          cfg.AWS.ECS.CreateEFS,
		EFSVolumeId:        cfg.AWS.ECS.EFSVolumeId,
		Mode:               cfg.AWS.ECS.Mode,
	}

	// Create ECS cluster
	if err := deployer.CreateCluster(ecsConfig.ClusterName); err != nil {
		fmt.Printf("ECS cluster might already exist: %v\n", err)
	}

	// Use advanced deployment if advanced features are enabled
	if ecsConfig.CreateSecrets || ecsConfig.CreateEFS {
		if err := deployer.DeployAdvanced(ecsConfig); err != nil {
			return fmt.Errorf("failed to deploy with advanced features: %w", err)
		}
	} else {
		// Basic deployment
		if err := deployer.CreateTaskDefinition(ecsConfig); err != nil {
			return fmt.Errorf("failed to create task definition: %w", err)
		}
	}

	// Create ECS service (for both advanced and basic deployments)
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

func runCleanup() error {
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
		DatabaseHTTPPort:   cfg.AWS.ECS.DatabaseHTTPPort,
		WebAppMemory:       cfg.AWS.ECS.WebAppMemory,
		WebAppCPU:          cfg.AWS.ECS.WebAppCPU,
		DatabaseMemory:     cfg.AWS.ECS.DatabaseMemory,
		DatabaseCPU:        cfg.AWS.ECS.DatabaseCPU,
		Environment:        cfg.AWS.ECS.Environment,
		CreateSecrets:      cfg.AWS.ECS.CreateSecrets,
		CreateEFS:          cfg.AWS.ECS.CreateEFS,
		EFSVolumeId:        cfg.AWS.ECS.EFSVolumeId,
		Mode:               cfg.AWS.ECS.Mode,
	}

	// Confirm cleanup with user
	fmt.Printf("This will delete the following resources:\n")
	fmt.Printf("  - ECS Service: %s\n", ecsConfig.ServiceName)
	fmt.Printf("  - ECS Cluster: %s (if empty)\n", ecsConfig.ClusterName)
	fmt.Printf("  - Task Definition: %s (all revisions)\n", ecsConfig.TaskDefinitionName)
	fmt.Printf("  - Load Balancer: %s-alb\n", ecsConfig.ServiceName)
	fmt.Printf("  - Target Group: %s-tg\n", ecsConfig.ServiceName)
	fmt.Printf("  - CloudWatch Log Groups\n")
	fmt.Print("\nAre you sure you want to proceed? (yes/no): ")

	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
		fmt.Println("Cleanup cancelled.")
		return nil
	}

	// Run cleanup
	if err := deployer.Cleanup(ecsConfig); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Println("✅ Cleanup completed successfully!")
	return nil
}
