package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
	"github.com/spf13/cobra"
	"opsagents/internal/config"
	"opsagents/pkg/builder"
	"opsagents/pkg/deploy"
	"opsagents/pkg/git"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "opsagents",
		Short: "OpsAgents - Build and Deploy Automation Tool",
		Long:  `A CLI tool to orchestrate building and deploying applications to AWS Lightsail`,
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build application and Docker images",
		Long:  `Pull from Git repository, build Go binary, and create Docker images`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting build process...")
			if err := runBuild(); err != nil {
				fmt.Printf("Build failed: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy to AWS Lightsail",
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

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runBuild() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize Git client
	gitClient := git.New(".")
	
	// Clone or pull repository
	fmt.Printf("Working with repository: %s\n", cfg.Git.Repository)
	if err := gitClient.CloneRepository(cfg.Git.Repository, cfg.Git.WorkingDir); err != nil {
		// If clone fails, try to pull instead
		if pullErr := gitClient.PullLatest(cfg.Git.WorkingDir); pullErr != nil {
			return fmt.Errorf("failed to clone and pull repository: %w", err)
		}
	}

	// Build Go binary
	sourceDir := fmt.Sprintf("%s/%s", cfg.Git.WorkingDir, cfg.Build.AppName)
	goBuilder := builder.NewGoBuilder(sourceDir, cfg.Build.OutputDir)
	
	if err := goBuilder.BuildBinary(cfg.Build.AppName); err != nil {
		return fmt.Errorf("failed to build Go binary: %w", err)
	}

	// Create Docker images
	dockerBuilder := builder.NewDockerBuilder(cfg.Git.WorkingDir)
	
	if err := dockerBuilder.CreateDockerfiles(); err != nil {
		return fmt.Errorf("failed to create Dockerfiles: %w", err)
	}

	appImageName := fmt.Sprintf("%s-app", cfg.Build.AppName)
	if err := dockerBuilder.BuildImage(appImageName, "Dockerfile.app"); err != nil {
		return fmt.Errorf("failed to build app Docker image: %w", err)
	}

	neo4jImageName := fmt.Sprintf("%s-neo4j", cfg.Build.AppName)
	if err := dockerBuilder.BuildImage(neo4jImageName, "Dockerfile.neo4j"); err != nil {
		return fmt.Errorf("failed to build Neo4j Docker image: %w", err)
	}

	fmt.Println("Build process completed successfully!")
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
		ImageName:     fmt.Sprintf("%s-app:latest", cfg.Build.AppName),
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