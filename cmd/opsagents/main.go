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

	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "opsagents",
		Short: "OpsAgents - Claude AI Agent for DevOps Automation",
		Long:  `An intelligent Claude AI agent that automates deploying pre-built applications to AWS Lightsail with natural language commands`,
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
		Short: "Deploy to AWS Lightsail (direct mode)",
		Long:  `Deploy Docker containers to AWS Lightsail`,
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

	// Initialize Lightsail deployer
	deployer, err := deploy.NewLightsailDeployer()
	if err != nil {
		return fmt.Errorf("failed to initialize Lightsail deployer: %w", err)
	}

	// Create container service configuration
	serviceConfig := deploy.ContainerServiceConfig{
		ServiceName:   cfg.AWS.Lightsail.ServiceName,
		Power:         types.ContainerServicePowerName(cfg.AWS.Lightsail.Power),
		Scale:         cfg.AWS.Lightsail.Scale,
		PublicDomain:  cfg.AWS.Lightsail.PublicDomain,
		ContainerName: cfg.AWS.Lightsail.ContainerName,
		ImageName:     cfg.Images.AppImage,
		Ports: map[string]int32{
			"8080": 8080,
		},
		Environment: cfg.AWS.Lightsail.Environment,
	}

	// Create container service
	if err := deployer.CreateContainerService(serviceConfig); err != nil {
		fmt.Printf("Container service might already exist, continuing with deployment: %v\n", err)
	}

	// Deploy container
	if err := deployer.DeployContainer(cfg.AWS.Lightsail.ServiceName, serviceConfig); err != nil {
		return fmt.Errorf("failed to deploy container: %w", err)
	}

	// Wait for service to be ready
	if err := deployer.WaitForServiceReady(cfg.AWS.Lightsail.ServiceName); err != nil {
		return fmt.Errorf("failed waiting for service to be ready: %w", err)
	}

	fmt.Println("Deployment completed successfully!")
	return nil
}
