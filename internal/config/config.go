package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AgentName string `mapstructure:"agent_name"`
	Port      int    `mapstructure:"port"`
	LogLevel  string `mapstructure:"log_level"`

	Git struct {
		Repository string `mapstructure:"repository"`
		Branch     string `mapstructure:"branch"`
		WorkingDir string `mapstructure:"working_dir"`
	} `mapstructure:"git"`

	Images struct {
		Registry   string `mapstructure:"registry"`
		AppImage   string `mapstructure:"app_image"`
		Neo4jImage string `mapstructure:"neo4j_image"`
	} `mapstructure:"images"`

	AWS struct {
		Region string `mapstructure:"region"`
		ECS    struct {
			ClusterName        string            `mapstructure:"cluster_name"`
			ServiceName        string            `mapstructure:"service_name"`
			TaskDefinitionName string            `mapstructure:"task_definition_name"`
			VpcId              string            `mapstructure:"vpc_id"`
			SubnetIds          []string          `mapstructure:"subnet_ids"`
			SecurityGroupIds   []string          `mapstructure:"security_group_ids"`
			LoadBalancerName   string            `mapstructure:"load_balancer_name"`
			WebAppPort         int32             `mapstructure:"webapp_port"`
			DatabasePort       int32             `mapstructure:"database_port"`      // Neo4j Bolt port (7687)
			DatabaseHTTPPort   int32             `mapstructure:"database_http_port"` // Neo4j HTTP port (7474)
			WebAppMemory       int32             `mapstructure:"webapp_memory"`
			WebAppCPU          int32             `mapstructure:"webapp_cpu"`
			DatabaseMemory     int32             `mapstructure:"database_memory"`
			DatabaseCPU        int32             `mapstructure:"database_cpu"`
			Environment        map[string]string `mapstructure:"environment"`
			CreateSecrets      bool              `mapstructure:"create_secrets"`
			CreateEFS          bool              `mapstructure:"create_efs"`
			EFSVolumeId        string            `mapstructure:"efs_volume_id"`
			Mode               string            `mapstructure:"mode"`
		} `mapstructure:"ecs"`
		Lightsail struct {
			ServiceName   string            `mapstructure:"service_name"`
			Power         string            `mapstructure:"power"`
			Scale         int32             `mapstructure:"scale"`
			PublicDomain  string            `mapstructure:"public_domain"`
			ContainerName string            `mapstructure:"container_name"`
			Environment   map[string]string `mapstructure:"environment"`
		} `mapstructure:"lightsail"`
	} `mapstructure:"aws"`

	Claude struct {
		Region      string  `mapstructure:"region"`
		ModelID     string  `mapstructure:"model_id"`
		Temperature float32 `mapstructure:"temperature"`
		MaxTokens   int     `mapstructure:"max_tokens"`
	} `mapstructure:"claude"`

	Auth struct {
		GitHubTokenEnv string `mapstructure:"github_token_env"`
		AWSProfileEnv  string `mapstructure:"aws_profile_env"`
	} `mapstructure:"auth"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("agent_name", "bigfootgolf-agent")
	viper.SetDefault("port", 8080)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("images.registry", "ghcr.io/jrzesz33")
	viper.SetDefault("images.app_image", "ghcr.io/jrzesz33/bigfootgolf-webapp:sha-788e856")
	viper.SetDefault("images.neo4j_image", "ghcr.io/jrzesz33/bigfootgolf-db:sha-9270da4")
	viper.SetDefault("aws.region", "us-east-1")
	// ECS defaults
	viper.SetDefault("aws.ecs.cluster_name", "bigfootgolf-cluster")
	viper.SetDefault("aws.ecs.service_name", "bigfootgolf-service")
	viper.SetDefault("aws.ecs.task_definition_name", "bigfootgolf-task")
	viper.SetDefault("aws.ecs.webapp_port", 8000)
	viper.SetDefault("aws.ecs.database_port", 7687)
	viper.SetDefault("aws.ecs.database_http_port", 7474)
	viper.SetDefault("aws.ecs.webapp_memory", 512)
	viper.SetDefault("aws.ecs.webapp_cpu", 256)
	viper.SetDefault("aws.ecs.database_memory", 512)
	viper.SetDefault("aws.ecs.database_cpu", 256)
	viper.SetDefault("aws.ecs.load_balancer_name", "bigfootgolf-alb")
	viper.SetDefault("aws.ecs.create_secrets", false)
	viper.SetDefault("aws.ecs.create_efs", false)
	viper.SetDefault("aws.ecs.mode", "prod")
	// Lightsail defaults (kept for compatibility)
	viper.SetDefault("aws.lightsail.service_name", "bigfootgolf-service")
	viper.SetDefault("aws.lightsail.power", "nano")
	viper.SetDefault("aws.lightsail.scale", 1)
	viper.SetDefault("aws.lightsail.container_name", "bigfootgolf-app")
	viper.SetDefault("claude.region", "us-east-1")
	viper.SetDefault("claude.model_id", "anthropic.claude-3-sonnet-20240229-v1:0")
	viper.SetDefault("claude.temperature", 0.1)
	viper.SetDefault("claude.max_tokens", 4096)
	viper.SetDefault("auth.github_token_env", "GITHUB_TOKEN")
	viper.SetDefault("auth.aws_profile_env", "AWS_PROFILE")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func CreateDefaultConfig() error {
	config := `agent_name: bigfootgolf-agent
port: 8080
log_level: info

images:
  registry: docker.io
  app_image: your-registry/bigfootgolf-app:latest
  neo4j_image: neo4j:5-community

aws:
  region: us-east-1
  ecs:
    cluster_name: bigfootgolf-cluster
    service_name: bigfootgolf-service
    task_definition_name: bigfootgolf-task
    vpc_id: ""  # Will be auto-detected or set via environment
    subnet_ids: []  # Will be auto-detected or set via environment
    security_group_ids: []  # Will be auto-detected or set via environment
    load_balancer_name: bigfootgolf-alb
    webapp_port: 8000
    database_port: 7687          # Neo4j Bolt protocol port
    database_http_port: 7474     # Neo4j HTTP interface port
    webapp_memory: 512
    webapp_cpu: 256
    database_memory: 512
    database_cpu: 256
    create_secrets: false      # Enable to create AWS Secrets Manager secrets
    create_efs: false         # Enable to create EFS volume for Neo4j persistence
    efs_volume_id: ""         # EFS Volume ID (auto-created if create_efs is true)
    mode: "prod"              # Application mode: prod, dev, test
    environment:
      ENV: production
      PORT: "8000"
  lightsail:
    service_name: bigfootgolf-service
    power: nano
    scale: 1
    public_domain: bigfootgolf.example.com
    container_name: bigfootgolf-app
    environment:
      ENV: production
      PORT: "8000"

claude:
  region: us-east-1
  model_id: anthropic.claude-3-sonnet-20240229-v1:0
  temperature: 0.1
  max_tokens: 4096

auth:
  github_token_env: GITHUB_TOKEN  # Environment variable for GitHub PAT
  aws_profile_env: AWS_PROFILE    # Environment variable for AWS profile
`

	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(config)); err != nil {
		return fmt.Errorf("failed to read default config: %w", err)
	}

	if err := viper.WriteConfigAs("config.yaml"); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("Default configuration created: config.yaml")
	return nil
}
