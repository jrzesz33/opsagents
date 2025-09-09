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
		Region    string `mapstructure:"region"`
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
	viper.SetDefault("images.registry", "docker.io")
	viper.SetDefault("images.app_image", "your-registry/bigfootgolf-app:latest")
	viper.SetDefault("images.neo4j_image", "neo4j:5-community")
	viper.SetDefault("aws.region", "us-east-1")
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
  lightsail:
    service_name: bigfootgolf-service
    power: nano
    scale: 1
    public_domain: bigfootgolf.example.com
    container_name: bigfootgolf-app
    environment:
      ENV: production
      PORT: "8080"

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
