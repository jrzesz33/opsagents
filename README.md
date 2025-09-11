# OpsAgents - Claude AI-Powered AWS ECS Deployment Tool

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![AWS](https://img.shields.io/badge/AWS-ECS-orange.svg)](https://aws.amazon.com/ecs/)
[![Claude](https://img.shields.io/badge/Claude-AI-purple.svg)](https://claude.ai)

An intelligent Claude AI agent that uses AWS Bedrock to automate deployment of containerized applications to AWS ECS (Elastic Container Service). Interact with your DevOps pipeline using natural language commands while the AI agent executes deployment workflows.

## ✨ Features

- **🤖 Claude AI Integration**: Natural language interaction with Claude AI via AWS Bedrock
- **🛠️ Intelligent Tools**: AI-powered deployment tools that understand context
- **🐳 External Image Support**: Deploy pre-built Docker images from external registries
- **☁️ AWS ECS Deployment**: Deploy containers to scalable AWS ECS with Fargate
- **🔒 Security First**: AWS Secrets Manager integration for secure credential management
- **💾 Persistence**: EFS storage for database persistence
- **📊 Monitoring**: CloudWatch logging and Application Load Balancer health checks
- **⚙️ Configuration Management**: Flexible YAML-based configuration with environment variable support
- **💬 Interactive Chat**: Chat with Claude to deploy and manage your applications
- **🧽 Easy Cleanup**: One-command resource removal

## 🚀 Quick Start

### 1. Build the Tool

```bash
go build -o build/opsagents cmd/opsagents/main.go
```

### 2. Set Environment Variables

```bash
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

This creates a `config.yaml` file with Claude AI, AWS Bedrock settings, and Docker image configuration.

### 4. Start the Claude AI Agent

```bash
./build/opsagents agent
```

Then interact with Claude naturally:

```
🤖 Claude OpsAgent - Your AI DevOps Assistant

You: deploy to production
🤖 Claude: I'll deploy the pre-built application to AWS ECS. The containers will be deployed with health checks and load balancing enabled.

You: check deployment status
🤖 Claude: Let me check the current deployment status for you...
```

### Alternative: Direct Commands

You can also use direct commands without the AI agent:

```bash
# Deploy directly  
./build/opsagents deploy

# Clean up all resources
./build/opsagents cleanup
```

## 📋 What Gets Deployed

### AWS Infrastructure
- **ECS Cluster** with Fargate capacity providers
- **ECS Service** with auto-scaling and health checks  
- **Application Load Balancer** with target groups
- **CloudWatch Log Groups** for application monitoring
- **VPC Integration** with auto-discovery
- **Security Groups** with least-privilege access

### Container Applications
- **Web Application** (configurable image, port 8000)
- **Neo4j Database** (ports 7474/7687, optional persistence)

### Advanced Features (Optional)
- **AWS Secrets Manager** - 6 auto-generated secrets
- **EFS Persistent Storage** - Database data persistence  
- **Multi-AZ Deployment** - High availability across subnets

## 🔧 Configuration Modes

### Basic Deployment
```yaml
aws:
  ecs:
    create_secrets: false
    create_efs: false
```
**Creates**: ECS service + Load balancer + 2 containers  
**Time**: ~3-5 minutes

### Advanced Deployment  
```yaml
aws:
  ecs:
    create_secrets: true    # Enables secure secrets
    create_efs: true       # Enables persistent storage
```
**Creates**: All basic resources + 6 secrets + EFS storage  
**Time**: ~5-8 minutes

## 🗂️ Container Environment Variables

### Web Application
| Variable | Value | Description |
|----------|-------|-------------|
| `MODE` | `prod` | Application mode |
| `DB_URI` | `bolt://localhost:7687` | Database connection |
| `DB_ADMIN` | *secret* | Database password (if secrets enabled) |
| `JWT_SECRET` | *secret* | Authentication key (if secrets enabled) |
| `SESSION_KEY` | *secret* | Session management (if secrets enabled) |

### Neo4j Database  
| Variable | Value | Description |
|----------|-------|-------------|
| `NEO4J_AUTH` | `none` | No auth (basic mode) |
| `NEO4J_PASSWORD` | *secret* | Secure auth (advanced mode) |

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
- Natural language commands for deployment operations
- Intelligent context understanding
- Automatic tool execution based on user intent
- Real-time status updates and feedback

### `opsagents deploy` (Direct Mode)
Deploys the application to AWS ECS:
- Creates ECS cluster, service, and task definition
- Deploys containers with Application Load Balancer
- Sets up health checks and auto-scaling
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

images:
  registry: docker.io
  app_image: your-registry/bigfootgolf-app:latest
  neo4j_image: neo4j:5-community

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

- **images.registry**: Docker registry URL (e.g., docker.io, gcr.io, your-private-registry.com)
- **images.app_image**: Full image name and tag for your application container
- **images.neo4j_image**: Neo4j database image (default: neo4j:5-community)
- **aws.lightsail.power**: Container size (nano, micro, small, medium, large)
- **aws.lightsail.scale**: Number of container instances
- **claude.region**: AWS region for Bedrock service
- **claude.model_id**: Claude model to use (Sonnet, Haiku, Opus)
- **claude.temperature**: Response creativity (0.0-1.0)
- **claude.max_tokens**: Maximum response length
- **auth.aws_profile_env**: Environment variable name for AWS profile

## 📋 Prerequisites

### Required Tools
- Go 1.24+ (for building the tool itself)
- AWS credentials (Access Keys or Profile)
- Pre-built Docker images in a registry

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

The tool deploys pre-built Docker images from external registries:

### Application Image
- Configured via `images.app_image` in config.yaml
- Should expose port 8080 for web applications
- Can be hosted on any Docker registry (Docker Hub, ECR, GCR, etc.)

### Neo4j Database Image
- Configured via `images.neo4j_image` in config.yaml
- Default: `neo4j:5-community` from Docker Hub
- Automatically configured with volume mounts for data persistence

## 🏗️ Architecture

### Claude AI Agent Flow
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   User Input    │───▶│  Claude AI      │───▶│  Tool Execution │
│                 │    │  (via Bedrock)  │    │                 │
│ • Natural Lang. │    │ • Intent        │    │ • deploy_app    │
│ • "Deploy"      │    │ • Tool Selection│    │ • get_status    │
│ • "Check status"│    │ • Context       │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Deployment Pipeline
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ External Images │───▶│ AWS Lightsail   │───▶│  Wait & Monitor │
│                 │    │                 │    │                 │
│ • App image     │    │ • Create service│    │ • Service state │
│ • Neo4j image   │    │ • Deploy images │    │ • Deployment    │
│ • From registry │    │ • Configure     │    │ • Health checks │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │   Service Ready │
                                              │                 │
                                              │ • Health checks │
                                              │ • Public URL    │
                                              └─────────────────┘
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
│   └── deploy/            # Deployment functionality
│       └── lightsail.go   # AWS Lightsail deployment
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

### Deployment Issues
- Verify AWS credentials are configured (`aws configure`)
- Check AWS region settings match your configuration
- Ensure Lightsail service limits aren't exceeded
- Verify container registry access for external images
- Check that specified Docker images exist and are accessible

### Authentication Issues
- **AWS Access Denied**: Check AWS credentials are configured (`aws sts get-caller-identity`)
- **Bedrock Model Access**: Ensure Claude models are enabled in AWS Bedrock console
- **Registry Access**: Verify credentials for private Docker registries

### Configuration Issues
- Run `opsagents config` to regenerate default configuration
- Check YAML syntax in `config.yaml`
- Verify environment variables are set correctly
- Ensure Docker image URLs are valid and accessible

## 🔒 Security

- **External Image Security**: Relies on pre-built, security-scanned images from trusted registries
- **Environment Variable Auth**: AWS credentials via environment variables
- **AWS IAM Best Practices**: Minimal required permissions for Bedrock and Lightsail
- **Secure AI Integration**: Claude AI runs in AWS Bedrock with enterprise security
- **Registry Security**: Supports private registries with authentication
- **No Hardcoded Secrets**: All credentials managed through environment variables

## 💰 Cost Optimization

- **Lightsail Nano Instances**: Most cost-effective container hosting (~$7/month)
- **Single Container**: Default single instance deployment
- **External Images**: No build infrastructure costs
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