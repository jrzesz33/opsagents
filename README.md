# OpsAgents - Claude AI Agent for DevOps Automation

An intelligent Claude AI agent that uses AWS Bedrock to automate building and deployment of applications to AWS Lightsail. Interact with your DevOps pipeline using natural language commands while the AI agent executes complex deployment workflows.

## ✨ Features

- **🤖 Claude AI Integration**: Natural language interaction with Claude AI via AWS Bedrock
- **🛠️ Intelligent Tools**: AI-powered build and deploy tools that understand context
- **📦 Git Repository Management**: Automatically clone and pull the latest changes from GitHub repositories
- **🔨 Go Binary Building**: Cross-compile Go applications with optimized settings
- **🐳 Docker Image Creation**: Build container images using secure Chainguard base images
- **☁️ AWS Lightsail Deployment**: Deploy containers to cost-efficient AWS Lightsail services
- **⚙️ Configuration Management**: Flexible YAML-based configuration with environment variable support
- **💬 Interactive Chat**: Chat with Claude to build, deploy, and manage your applications

## 🚀 Quick Start

### 1. Build the Tool

```bash
go build -o build/opsagents cmd/opsagents/main.go
```

### 2. Set Environment Variables

```bash
# GitHub Personal Access Token (for private repos)
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"

# AWS Credentials for Bedrock and Lightsail
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"

# Optional: AWS Profile (alternative to access keys)
export AWS_PROFILE="your-aws-profile"
```

### 3. Generate Configuration

```bash
./build/opsagents config
```

This creates a `config.yaml` file with Claude AI and AWS Bedrock settings.

### 4. Start the Claude AI Agent

```bash
./build/opsagents agent
```

Then interact with Claude naturally:

```
🤖 Claude OpsAgent - Your AI DevOps Assistant

You: build the application
🤖 Claude: I'll build the application for you. Let me clone the repository, build the Go binary, and create Docker images.

You: deploy to production
🤖 Claude: I'll deploy the application to AWS Lightsail. The containers will be deployed with health checks enabled.

You: check deployment status
🤖 Claude: Let me check the current deployment status for you...
```

### Alternative: Direct Commands

You can also use direct commands without the AI agent:

```bash
# Build directly
./build/opsagents build

# Deploy directly  
./build/opsagents deploy
```

### Environment Setup Helper

Copy the example environment file and customize:

```bash
cp .env.example .env
# Edit .env with your actual tokens and credentials
# Then source it: source .env
```

## 📋 Commands

### `opsagents agent`
**Start the Claude AI Agent** - Interactive chat interface with Claude AI:
- Natural language commands for build and deploy operations
- Intelligent context understanding
- Automatic tool execution based on user intent
- Real-time status updates and feedback

### `opsagents build` (Direct Mode)
Executes the complete build pipeline:
- Clones/pulls from the configured Git repository
- Builds the Go binary with cross-compilation (Linux/AMD64)
- Creates Dockerfiles using Chainguard base images
- Builds Docker images for both the application and Neo4j database

### `opsagents deploy` (Direct Mode)
Deploys the application to AWS Lightsail:
- Creates or updates the Lightsail container service
- Deploys the application container with health checks
- Waits for the service to become ready
- Provides the service URL when deployment is complete

### `opsagents config`
Generates a default `config.yaml` file with Claude AI and AWS Bedrock configuration.

## ⚙️ Configuration

The tool uses a `config.yaml` file for configuration. Here's the structure:

```yaml
agent_name: bigfootgolf-agent
port: 8080
log_level: info

git:
  repository: https://github.com/jrzesz33/bigfootgolf
  branch: main
  working_dir: ./workspace

build:
  app_name: bigfootgolf
  output_dir: ./build

aws:
  region: us-east-1
  lightsail:
    service_name: bigfootgolf-service
    power: nano          # Most cost-efficient option
    scale: 1
    public_domain: bigfootgolf.example.com
    container_name: bigfootgolf-app
    environment:
      ENV: production
      PORT: "8080"

claude:                  # Claude AI configuration
  region: us-east-1     # AWS region for Bedrock
  model_id: anthropic.claude-3-sonnet-20240229-v1:0
  temperature: 0.1      # Lower for more focused responses
  max_tokens: 4096      # Maximum response length

auth:                    # Authentication configuration
  github_token_env: GITHUB_TOKEN  # Environment variable for GitHub PAT
  aws_profile_env: AWS_PROFILE    # Environment variable for AWS profile
```

### Configuration Options

- **git.repository**: GitHub repository URL to clone
- **git.branch**: Git branch to use (default: main)
- **git.working_dir**: Local directory for cloned repository
- **build.app_name**: Name of the Go binary and Docker images
- **aws.lightsail.power**: Container size (nano, micro, small, medium, large)
- **aws.lightsail.scale**: Number of container instances
- **claude.region**: AWS region for Bedrock service
- **claude.model_id**: Claude model to use (Sonnet, Haiku, Opus)
- **claude.temperature**: Response creativity (0.0-1.0)
- **claude.max_tokens**: Maximum response length
- **auth.github_token_env**: Environment variable name for GitHub Personal Access Token
- **auth.aws_profile_env**: Environment variable name for AWS profile

## 📋 Prerequisites

### Required Tools
- Go 1.24+
- Docker  
- Git
- GitHub Personal Access Token (for private repositories)
- AWS credentials (Access Keys or Profile)

### AWS Permissions
Your AWS credentials need the following permissions:

**AWS Bedrock (for Claude AI):**
- `bedrock:InvokeModel`
- `bedrock:ListFoundationModels`

**AWS Lightsail (for deployment):**
- `lightsail:CreateContainerService`
- `lightsail:CreateContainerServiceDeployment`
- `lightsail:GetContainerServices`

### Authentication Setup

#### GitHub Personal Access Token
1. Go to GitHub Settings > Developer settings > Personal access tokens
2. Generate new token (classic) with `repo` scope for private repositories
3. Set as environment variable: `export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"`

#### AWS Credentials
**Option 1: Environment Variables**
```bash
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
export AWS_DEFAULT_REGION="us-east-1"
```

**Option 2: AWS Profile**
```bash
aws configure --profile opsagents
export AWS_PROFILE="opsagents"
```

#### AWS Bedrock Model Access
1. Go to AWS Bedrock console
2. Navigate to "Model access"
3. Request access to Anthropic Claude models
4. Wait for approval (usually instant for Claude 3 Sonnet)

## Docker Images

The tool creates two Docker images:

### Application Image (`bigfootgolf-app`)
- Base: `cgr.dev/chainguard/static:latest` (minimal, secure)
- Contains the compiled Go binary
- Exposes port 8080

### Neo4j Database Image (`bigfootgolf-neo4j`)
- Base: `neo4j:5-community`
- Pre-configured with authentication and database settings
- Includes volume mounts for data persistence

## 🏗️ Architecture

### Claude AI Agent Flow
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   User Input    │───▶│  Claude AI      │───▶│  Tool Execution │
│                 │    │  (via Bedrock)  │    │                 │
│ • Natural Lang. │    │ • Intent        │    │ • build_app     │
│ • "Build app"   │    │ • Tool Selection│    │ • deploy_app    │
│ • "Deploy"      │    │ • Context       │    │ • get_status    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Build and Deploy Pipeline
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Git Clone     │───▶│   Go Build      │───▶│  Docker Build   │
│                 │    │                 │    │                 │
│ • Clone repo    │    │ • Cross-compile │    │ • App image     │
│ • Pull latest   │    │ • Linux/AMD64   │    │ • Neo4j image   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Service Ready │◀───│  Wait & Monitor │◀───│ AWS Lightsail   │
│                 │    │                 │    │                 │
│ • Health checks │    │ • Service state │    │ • Create service│
│ • Public URL    │    │ • Deployment    │    │ • Deploy images │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Project Structure

```
opsagents/
├── cmd/opsagents/          # CLI application entry point
│   └── main.go            # Main CLI with Claude agent integration
├── pkg/                    # Public packages
│   ├── agent/             # Claude AI agent
│   │   ├── agent.go       # Base agent interface
│   │   └── claude.go      # Claude AI implementation with Bedrock
│   ├── builder/           # Build functionality
│   │   ├── docker.go      # Docker image building
│   │   └── golang.go      # Go binary building
│   ├── deploy/            # Deployment functionality
│   │   └── lightsail.go   # AWS Lightsail deployment
│   └── git/               # Git repository management
│       └── clone.go       # Git clone and pull operations
├── internal/              # Private packages
│   └── config/            # Configuration management
│       └── config.go      # YAML config with Claude/Bedrock settings
├── docs/                  # Documentation
│   ├── README.md         # Project overview
│   └── reqs/             # Requirements specifications
├── .devcontainer/        # Development container config
├── .github/              # GitHub workflows
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
└── config.yaml           # Runtime configuration with Claude AI settings
```

## Development

### Running in Development Container

This project includes a devcontainer configuration for consistent development:

```bash
# Open in VS Code with Dev Containers extension
code .
```

The devcontainer includes:
- Go 1.24
- AWS CLI
- Docker-in-Docker
- GitHub CLI

### Building from Source

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o build/opsagents cmd/opsagents/main.go

# Build with optimizations
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o build/opsagents cmd/opsagents/main.go
```

## Troubleshooting

### Build Issues
- Ensure Docker is running and accessible
- Verify Git repository access and credentials
- Check Go version compatibility (1.24+)

### Deployment Issues
- Verify AWS credentials are configured (`aws configure`)
- Check AWS region settings match your configuration
- Ensure Lightsail service limits aren't exceeded
- Verify container registry access for base images

### Authentication Issues
- **Git Clone Fails**: Verify `GITHUB_TOKEN` environment variable is set
- **AWS Access Denied**: Check AWS credentials are configured (`aws sts get-caller-identity`)
- **Bedrock Model Access**: Ensure Claude models are enabled in AWS Bedrock console
- **Private Repository Access**: GitHub PAT needs `repo` scope for private repositories

### Configuration Issues
- Run `opsagents config` to regenerate default configuration
- Check YAML syntax in `config.yaml`
- Verify environment variables are set correctly

## 🔒 Security

- **Secure Base Images**: Uses Chainguard distroless images for minimal attack surface
- **Environment Variable Auth**: GitHub PAT and AWS credentials via environment variables
- **AWS IAM Best Practices**: Minimal required permissions for Bedrock and Lightsail
- **Secure AI Integration**: Claude AI runs in AWS Bedrock with enterprise security
- **Token-based Git Access**: GitHub Personal Access Tokens for secure repository access
- **No Hardcoded Secrets**: All credentials managed through environment variables

## 💰 Cost Optimization

- **Lightsail Nano Instances**: Most cost-effective container hosting (~$7/month)
- **Single Container**: Default single instance deployment
- **Minimal Base Images**: Chainguard images reduce storage and transfer costs
- **Bedrock Pay-per-Use**: Only pay for Claude AI interactions
- **Intelligent Scaling**: AI agent can optimize resource usage based on needs

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- Create an issue in the GitHub repository
- Check the troubleshooting section above
- Review the AWS Lightsail documentation for deployment-specific issues