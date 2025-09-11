package deploy

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type ECSDeployer struct {
	ecsClient     *ecs.Client
	ec2Client     *ec2.Client
	elbv2Client   *elasticloadbalancingv2.Client
	iamClient     *iam.Client
	logsClient    *cloudwatchlogs.Client
	secretsClient *secretsmanager.Client
	efsClient     *efs.Client
	ctx           context.Context
}

type ECSConfig struct {
	ClusterName        string
	ServiceName        string
	TaskDefinitionName string
	VpcId              string
	SubnetIds          []string
	SecurityGroupIds   []string
	LoadBalancerName   string
	WebAppImage        string
	DatabaseImage      string
	WebAppPort         int32
	DatabasePort       int32 // Neo4j Bolt port (7687)
	DatabaseHTTPPort   int32 // Neo4j HTTP port (7474)
	WebAppMemory       int32
	WebAppCPU          int32
	DatabaseMemory     int32
	DatabaseCPU        int32
	Environment        map[string]string
	// New configuration options
	CreateSecrets bool
	CreateEFS     bool
	EFSVolumeId   string
	Mode          string
}

func NewECSDeployer() (*ECSDeployer, error) {
	// Load AWS config with explicit environment variable credentials
	var cfg aws.Config
	var err error

	// Check if we have environment variables for AWS credentials
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")

	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
		if region == "" {
			region = "us-east-1" // Default region
		}
	}

	if accessKey != "" && secretKey != "" {
		// Use static credentials from environment variables and disable IMDS
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
			config.WithEC2IMDSClientEnableState(imds.ClientDisabled),
		)
	} else {
		// Fall back to default credential chain (excluding IMDS)
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithEC2IMDSClientEnableState(imds.ClientDisabled),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &ECSDeployer{
		ecsClient:     ecs.NewFromConfig(cfg),
		ec2Client:     ec2.NewFromConfig(cfg),
		elbv2Client:   elasticloadbalancingv2.NewFromConfig(cfg),
		iamClient:     iam.NewFromConfig(cfg),
		logsClient:    cloudwatchlogs.NewFromConfig(cfg),
		secretsClient: secretsmanager.NewFromConfig(cfg),
		efsClient:     efs.NewFromConfig(cfg),
		ctx:           context.Background(),
	}, nil
}

func (d *ECSDeployer) CreateCluster(clusterName string) error {
	fmt.Printf("Creating ECS cluster: %s\n", clusterName)

	input := &ecs.CreateClusterInput{
		ClusterName:       aws.String(clusterName),
		CapacityProviders: []string{"FARGATE", "FARGATE_SPOT"},
		DefaultCapacityProviderStrategy: []types.CapacityProviderStrategyItem{
			{
				CapacityProvider: aws.String("FARGATE"),
				Weight:           1,
			},
		},
	}

	_, err := d.ecsClient.CreateCluster(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create ECS cluster: %w", err)
	}

	fmt.Printf("ECS cluster %s created successfully\n", clusterName)
	return nil
}

func (d *ECSDeployer) CreateTaskDefinition(config ECSConfig) error {
	fmt.Printf("Creating task definition: %s\n", config.TaskDefinitionName)

	// Create CloudWatch log groups
	webAppLogGroup := fmt.Sprintf("/ecs/%s-webapp", config.TaskDefinitionName)
	dbLogGroup := fmt.Sprintf("/ecs/%s-database", config.TaskDefinitionName)

	d.createLogGroup(webAppLogGroup)
	d.createLogGroup(dbLogGroup)

	// Task execution role ARN - this should exist or be created
	taskExecutionRoleArn := "arn:aws:iam::" + d.getAccountId() + ":role/ecsTaskExecutionRole"

	containerDefinitions := []types.ContainerDefinition{
		{
			Name:   aws.String("webapp"),
			Image:  aws.String(config.WebAppImage),
			Memory: aws.Int32(config.WebAppMemory),
			Cpu:    config.WebAppCPU,
			PortMappings: []types.PortMapping{
				{
					ContainerPort: aws.Int32(config.WebAppPort),
					Protocol:      types.TransportProtocolTcp,
				},
			},
			Environment: d.mapToEnvironment(config.Environment),
			LogConfiguration: &types.LogConfiguration{
				LogDriver: types.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         webAppLogGroup,
					"awslogs-region":        os.Getenv("AWS_REGION"),
					"awslogs-stream-prefix": "ecs",
				},
			},
			Essential: aws.Bool(true),
		},
		{
			Name:   aws.String("database"),
			Image:  aws.String(config.DatabaseImage),
			Memory: aws.Int32(config.DatabaseMemory),
			Cpu:    config.DatabaseCPU,
			PortMappings: []types.PortMapping{
				{
					ContainerPort: aws.Int32(7474), // Neo4j HTTP interface
					Protocol:      types.TransportProtocolTcp,
				},
				{
					ContainerPort: aws.Int32(7687), // Neo4j Bolt protocol
					Protocol:      types.TransportProtocolTcp,
				},
			},
			Environment: []types.KeyValuePair{
				{
					Name:  aws.String("NEO4J_AUTH"),
					Value: aws.String("none"),
				},
			},
			LogConfiguration: &types.LogConfiguration{
				LogDriver: types.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         dbLogGroup,
					"awslogs-region":        os.Getenv("AWS_REGION"),
					"awslogs-stream-prefix": "ecs",
				},
			},
			Essential: aws.Bool(false),
		},
	}

	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(config.TaskDefinitionName),
		NetworkMode:             types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		Cpu:                     aws.String(fmt.Sprintf("%d", config.WebAppCPU+config.DatabaseCPU)),
		Memory:                  aws.String(fmt.Sprintf("%d", config.WebAppMemory+config.DatabaseMemory)),
		ExecutionRoleArn:        aws.String(taskExecutionRoleArn),
		ContainerDefinitions:    containerDefinitions,
	}

	_, err := d.ecsClient.RegisterTaskDefinition(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to register task definition: %w", err)
	}

	fmt.Printf("Task definition %s registered successfully\n", config.TaskDefinitionName)
	return nil
}

func (d *ECSDeployer) CreateTaskDefinitionAdvanced(config ECSConfig, secretArns map[string]string) error {
	fmt.Printf("Creating advanced task definition: %s\n", config.TaskDefinitionName)

	// Create CloudWatch log groups
	webAppLogGroup := fmt.Sprintf("/ecs/%s-webapp", config.TaskDefinitionName)
	dbLogGroup := fmt.Sprintf("/ecs/%s-database", config.TaskDefinitionName)

	d.createLogGroup(webAppLogGroup)
	d.createLogGroup(dbLogGroup)

	// Task execution role ARN - this should exist or be created
	taskExecutionRoleArn := "arn:aws:iam::" + d.getAccountId() + ":role/ecsTaskExecutionRole"

	// Build web app environment variables and secrets
	webAppEnv := d.mapToEnvironment(config.Environment)
	webAppSecrets := []types.Secret{}

	// Add MODE environment variable
	webAppEnv = append(webAppEnv, types.KeyValuePair{
		Name:  aws.String("MODE"),
		Value: aws.String(config.Mode),
	})

	// Map secrets to environment variables for web app
	if arn, exists := secretArns["DB_SECRET_ARN"]; exists {
		webAppSecrets = append(webAppSecrets, types.Secret{
			Name:      aws.String("DB_ADMIN"),
			ValueFrom: aws.String(arn),
		})
	}
	if arn, exists := secretArns["JWT_SECRET_ARN"]; exists {
		webAppSecrets = append(webAppSecrets, types.Secret{
			Name:      aws.String("JWT_SECRET"),
			ValueFrom: aws.String(arn),
		})
	}
	if arn, exists := secretArns["SESSION_KEY_ARN"]; exists {
		webAppSecrets = append(webAppSecrets, types.Secret{
			Name:      aws.String("SESSION_KEY"),
			ValueFrom: aws.String(arn),
		})
	}
	if arn, exists := secretArns["ANTHROPIC_SECRET_ARN"]; exists {
		webAppSecrets = append(webAppSecrets, types.Secret{
			Name:      aws.String("ANTHROPIC_API_KEY"),
			ValueFrom: aws.String(arn),
		})
	}
	if arn, exists := secretArns["GMAIL_USER_ARN"]; exists {
		webAppSecrets = append(webAppSecrets, types.Secret{
			Name:      aws.String("GMAIL_USER"),
			ValueFrom: aws.String(arn),
		})
	}
	if arn, exists := secretArns["GMAIL_PASS_ARN"]; exists {
		webAppSecrets = append(webAppSecrets, types.Secret{
			Name:      aws.String("GMAIL_PASS"),
			ValueFrom: aws.String(arn),
		})
	}

	// Build database environment variables and secrets
	dbSecrets := []types.Secret{}
	if arn, exists := secretArns["DB_SECRET_ARN"]; exists {
		dbSecrets = append(dbSecrets, types.Secret{
			Name:      aws.String("NEO4J_PASSWORD"),
			ValueFrom: aws.String(arn),
		})
	}

	// Container definitions
	containerDefinitions := []types.ContainerDefinition{
		{
			Name:   aws.String("webapp"),
			Image:  aws.String(config.WebAppImage),
			Memory: aws.Int32(config.WebAppMemory),
			Cpu:    config.WebAppCPU,
			PortMappings: []types.PortMapping{
				{
					ContainerPort: aws.Int32(config.WebAppPort),
					Protocol:      types.TransportProtocolTcp,
				},
			},
			Environment: webAppEnv,
			Secrets:     webAppSecrets,
			LogConfiguration: &types.LogConfiguration{
				LogDriver: types.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         webAppLogGroup,
					"awslogs-region":        os.Getenv("AWS_REGION"),
					"awslogs-stream-prefix": "ecs",
				},
			},
			Essential: aws.Bool(true),
		},
		{
			Name:   aws.String("database"),
			Image:  aws.String(config.DatabaseImage),
			Memory: aws.Int32(config.DatabaseMemory),
			Cpu:    config.DatabaseCPU,
			PortMappings: []types.PortMapping{
				{
					ContainerPort: aws.Int32(7474), // Neo4j HTTP
					Protocol:      types.TransportProtocolTcp,
				},
				{
					ContainerPort: aws.Int32(7687), // Neo4j Bolt
					Protocol:      types.TransportProtocolTcp,
				},
			},
			Secrets: dbSecrets,
			LogConfiguration: &types.LogConfiguration{
				LogDriver: types.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         dbLogGroup,
					"awslogs-region":        os.Getenv("AWS_REGION"),
					"awslogs-stream-prefix": "ecs",
				},
			},
			Essential: aws.Bool(false),
		},
	}

	// Add EFS mount if configured
	var volumes []types.Volume
	if config.CreateEFS && config.EFSVolumeId != "" {
		volumes = []types.Volume{
			{
				Name: aws.String("neo4j-data"),
				EfsVolumeConfiguration: &types.EFSVolumeConfiguration{
					FileSystemId:      aws.String(config.EFSVolumeId),
					RootDirectory:     aws.String("/"),
					TransitEncryption: types.EFSTransitEncryptionEnabled,
				},
			},
		}

		// Add mount point to database container
		for i := range containerDefinitions {
			if *containerDefinitions[i].Name == "database" {
				containerDefinitions[i].MountPoints = []types.MountPoint{
					{
						SourceVolume:  aws.String("neo4j-data"),
						ContainerPath: aws.String("/data"),
						ReadOnly:      aws.Bool(false),
					},
				}
				break
			}
		}
	}

	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(config.TaskDefinitionName),
		NetworkMode:             types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		Cpu:                     aws.String(fmt.Sprintf("%d", config.WebAppCPU+config.DatabaseCPU)),
		Memory:                  aws.String(fmt.Sprintf("%d", config.WebAppMemory+config.DatabaseMemory)),
		ExecutionRoleArn:        aws.String(taskExecutionRoleArn),
		ContainerDefinitions:    containerDefinitions,
		Volumes:                 volumes,
	}

	_, err := d.ecsClient.RegisterTaskDefinition(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to register task definition: %w", err)
	}

	fmt.Printf("Advanced task definition %s registered successfully\n", config.TaskDefinitionName)
	return nil
}

// DeployAdvanced handles the full deployment with secrets and EFS
func (d *ECSDeployer) DeployAdvanced(config ECSConfig) error {
	fmt.Printf("Starting advanced deployment for service: %s\n", config.ServiceName)

	// Create secrets if enabled
	var secretArns map[string]string
	if config.CreateSecrets {
		var err error
		secretArns, err = d.CreateSecrets(config.ServiceName)
		if err != nil {
			return fmt.Errorf("failed to create secrets: %w", err)
		}
	}

	// Create EFS if enabled
	if config.CreateEFS {
		// Auto-discover VPC and subnets if not provided
		if config.VpcId == "" || len(config.SubnetIds) == 0 {
			updatedConfig, err := d.autoDiscoverNetworking(config)
			if err != nil {
				return fmt.Errorf("failed to auto-discover VPC config: %w", err)
			}
			config = updatedConfig
		}

		efsId, err := d.CreateEFS(config.ServiceName, config.SubnetIds, config.SecurityGroupIds)
		if err != nil {
			return fmt.Errorf("failed to create EFS: %w", err)
		}
		config.EFSVolumeId = efsId
		fmt.Printf("EFS Volume ID set to: %s\n", efsId)
	}

	// Create task definition (advanced or basic)
	if config.CreateSecrets && secretArns != nil {
		if err := d.CreateTaskDefinitionAdvanced(config, secretArns); err != nil {
			return fmt.Errorf("failed to create advanced task definition: %w", err)
		}
	} else {
		if err := d.CreateTaskDefinition(config); err != nil {
			return fmt.Errorf("failed to create task definition: %w", err)
		}
	}

	return nil
}

func (d *ECSDeployer) CreateService(config ECSConfig) error {
	fmt.Printf("Creating ECS service: %s\n", config.ServiceName)

	// Check if service already exists
	describeInput := &ecs.DescribeServicesInput{
		Cluster:  aws.String(config.ClusterName),
		Services: []string{config.ServiceName},
	}

	describeOutput, err := d.ecsClient.DescribeServices(d.ctx, describeInput)
	if err == nil && len(describeOutput.Services) > 0 {
		service := describeOutput.Services[0]
		if *service.Status == "ACTIVE" {
			fmt.Printf("ECS service %s already exists and is active, updating task definition\n", config.ServiceName)
			// Update the service with the new task definition
			_, updateErr := d.ecsClient.UpdateService(d.ctx, &ecs.UpdateServiceInput{
				Cluster:        aws.String(config.ClusterName),
				Service:        aws.String(config.ServiceName),
				TaskDefinition: aws.String(config.TaskDefinitionName),
			})
			if updateErr != nil {
				return fmt.Errorf("failed to update ECS service: %w", updateErr)
			}
			fmt.Printf("ECS service %s updated successfully\n", config.ServiceName)
			return nil
		}
	}

	// Auto-discover VPC and subnets if not provided
	if config.VpcId == "" || len(config.SubnetIds) == 0 {
		config, err = d.autoDiscoverNetworking(config)
		if err != nil {
			return fmt.Errorf("failed to auto-discover networking: %w", err)
		}
	}

	// Create load balancer first
	loadBalancerArn, err := d.createLoadBalancer(config)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Create target group for load balancer
	targetGroupArn, err := d.createTargetGroup(config)
	if err != nil {
		return fmt.Errorf("failed to create target group: %w", err)
	}

	// Create listener to connect load balancer to target group
	err = d.createListener(loadBalancerArn, targetGroupArn, config)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	input := &ecs.CreateServiceInput{
		ServiceName:    aws.String(config.ServiceName),
		Cluster:        aws.String(config.ClusterName),
		TaskDefinition: aws.String(config.TaskDefinitionName),
		DesiredCount:   aws.Int32(1),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        config.SubnetIds,
				SecurityGroups: config.SecurityGroupIds,
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		},
		LoadBalancers: []types.LoadBalancer{
			{
				TargetGroupArn: aws.String(targetGroupArn),
				ContainerName:  aws.String("webapp"),
				ContainerPort:  aws.Int32(config.WebAppPort),
			},
		},
	}

	_, err = d.ecsClient.CreateService(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create ECS service: %w", err)
	}

	fmt.Printf("ECS service %s created successfully\n", config.ServiceName)
	return nil
}

func (d *ECSDeployer) GetServiceStatus(clusterName, serviceName string) (string, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}

	output, err := d.ecsClient.DescribeServices(d.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe service: %w", err)
	}

	if len(output.Services) == 0 {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	service := output.Services[0]
	status := fmt.Sprintf("Service: %s\nStatus: %s\nRunning: %d\nPending: %d\nDesired: %d",
		*service.ServiceName,
		*service.Status,
		service.RunningCount,
		service.PendingCount,
		service.DesiredCount)

	return status, nil
}

func (d *ECSDeployer) WaitForServiceStable(clusterName, serviceName string) error {
	fmt.Printf("Waiting for service %s to be stable...\n", serviceName)

	waiter := ecs.NewServicesStableWaiter(d.ecsClient)
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}

	err := waiter.Wait(d.ctx, input, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("failed waiting for service to be stable: %w", err)
	}

	fmt.Printf("Service %s is now stable!\n", serviceName)
	return nil
}

func (d *ECSDeployer) createTargetGroup(config ECSConfig) (string, error) {
	targetGroupName := fmt.Sprintf("%s-tg", config.ServiceName)
	fmt.Printf("Creating target group for service: %s\n", config.ServiceName)

	// First, check if a target group with this name already exists
	describeInput := &elasticloadbalancingv2.DescribeTargetGroupsInput{
		Names: []string{targetGroupName},
	}

	describeOutput, err := d.elbv2Client.DescribeTargetGroups(d.ctx, describeInput)
	if err == nil && len(describeOutput.TargetGroups) > 0 {
		// Target group exists, reuse it regardless of settings
		// This avoids the complexity of deleting and recreating
		existingTG := describeOutput.TargetGroups[0]
		fmt.Printf("Target group %s already exists, reusing it (port: %d, protocol: %s)\n", 
			targetGroupName, *existingTG.Port, existingTG.Protocol)
		return *existingTG.TargetGroupArn, nil
	}

	// Create new target group
	input := &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:                       aws.String(targetGroupName),
		Protocol:                   elbv2types.ProtocolEnumHttp,
		Port:                       aws.Int32(config.WebAppPort),
		VpcId:                      aws.String(config.VpcId),
		TargetType:                 elbv2types.TargetTypeEnumIp,
		HealthCheckProtocol:        elbv2types.ProtocolEnumHttp,
		HealthCheckPath:            aws.String("/health"),
		HealthCheckIntervalSeconds: aws.Int32(30),
		HealthyThresholdCount:      aws.Int32(2),
		UnhealthyThresholdCount:    aws.Int32(3),
	}

	output, err := d.elbv2Client.CreateTargetGroup(d.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create target group: %w", err)
	}

	fmt.Printf("Created target group: %s\n", targetGroupName)
	return *output.TargetGroups[0].TargetGroupArn, nil
}

func (d *ECSDeployer) createLogGroup(logGroupName string) error {
	input := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroupName),
	}

	_, err := d.logsClient.CreateLogGroup(d.ctx, input)
	if err != nil {
		// Log group might already exist, which is fine
		fmt.Printf("Log group %s might already exist: %v\n", logGroupName, err)
	}

	return nil
}

func (d *ECSDeployer) mapToEnvironment(envMap map[string]string) []types.KeyValuePair {
	var env []types.KeyValuePair
	for key, value := range envMap {
		env = append(env, types.KeyValuePair{
			Name:  aws.String(key),
			Value: aws.String(value),
		})
	}
	return env
}

func (d *ECSDeployer) getAccountId() string {
	// This is a placeholder - in a real implementation, you'd get this from STS
	// For now, return the account ID from your .env or a default
	return "944945738659" // Replace with actual account ID retrieval
}

func (d *ECSDeployer) autoDiscoverNetworking(config ECSConfig) (ECSConfig, error) {
	fmt.Println("Auto-discovering VPC and subnet configuration...")

	// Get default VPC
	vpcResult, err := d.ec2Client.DescribeVpcs(d.ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return config, fmt.Errorf("failed to describe VPCs: %w", err)
	}

	if len(vpcResult.Vpcs) == 0 {
		return config, fmt.Errorf("no default VPC found")
	}

	defaultVpc := vpcResult.Vpcs[0]
	config.VpcId = *defaultVpc.VpcId
	fmt.Printf("Found default VPC: %s\n", config.VpcId)

	// Get public subnets from the default VPC
	subnetResult, err := d.ec2Client.DescribeSubnets(d.ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{config.VpcId},
			},
		},
	})
	if err != nil {
		return config, fmt.Errorf("failed to describe subnets: %w", err)
	}

	config.SubnetIds = []string{}
	for _, subnet := range subnetResult.Subnets {
		config.SubnetIds = append(config.SubnetIds, *subnet.SubnetId)
	}

	if len(config.SubnetIds) == 0 {
		return config, fmt.Errorf("no subnets found in VPC %s", config.VpcId)
	}

	fmt.Printf("Found %d subnets: %v\n", len(config.SubnetIds), config.SubnetIds)

	// Create or get default security group if not provided
	if len(config.SecurityGroupIds) == 0 {
		sgResult, err := d.ec2Client.DescribeSecurityGroups(d.ctx, &ec2.DescribeSecurityGroupsInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{config.VpcId},
				},
				{
					Name:   aws.String("group-name"),
					Values: []string{"default"},
				},
			},
		})
		if err != nil {
			return config, fmt.Errorf("failed to describe security groups: %w", err)
		}

		if len(sgResult.SecurityGroups) > 0 {
			config.SecurityGroupIds = []string{*sgResult.SecurityGroups[0].GroupId}
			fmt.Printf("Using default security group: %s\n", config.SecurityGroupIds[0])
		}
	}

	return config, nil
}

func (d *ECSDeployer) createLoadBalancer(config ECSConfig) (string, error) {
	loadBalancerName := fmt.Sprintf("%s-alb", config.ServiceName)
	fmt.Printf("Creating load balancer for service: %s\n", config.ServiceName)

	// First, check if a load balancer with this name already exists
	describeInput := &elasticloadbalancingv2.DescribeLoadBalancersInput{
		Names: []string{loadBalancerName},
	}

	describeOutput, err := d.elbv2Client.DescribeLoadBalancers(d.ctx, describeInput)
	if err == nil && len(describeOutput.LoadBalancers) > 0 {
		// Load balancer exists, check if it's available and reuse it
		existingLB := describeOutput.LoadBalancers[0]
		if existingLB.State.Code == elbv2types.LoadBalancerStateEnumActive {
			fmt.Printf("Load balancer %s already exists and is active, reusing it\n", loadBalancerName)
			return *existingLB.LoadBalancerArn, nil
		}
	}

	// Create new load balancer
	input := &elasticloadbalancingv2.CreateLoadBalancerInput{
		Name:           aws.String(loadBalancerName),
		Subnets:        config.SubnetIds,
		SecurityGroups: config.SecurityGroupIds,
		Scheme:         elbv2types.LoadBalancerSchemeEnumInternetFacing,
		Type:           elbv2types.LoadBalancerTypeEnumApplication,
		IpAddressType:  elbv2types.IpAddressTypeIpv4,
	}

	output, err := d.elbv2Client.CreateLoadBalancer(d.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create load balancer: %w", err)
	}

	loadBalancerArn := *output.LoadBalancers[0].LoadBalancerArn
	fmt.Printf("Load balancer created: %s\n", loadBalancerArn)

	return loadBalancerArn, nil
}

func (d *ECSDeployer) createListener(loadBalancerArn, targetGroupArn string, _ ECSConfig) error {
	fmt.Printf("Creating listener for load balancer\n")

	// First, check if a listener already exists for this load balancer on port 80
	describeInput := &elasticloadbalancingv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
	}

	describeOutput, err := d.elbv2Client.DescribeListeners(d.ctx, describeInput)
	if err == nil && len(describeOutput.Listeners) > 0 {
		for _, listener := range describeOutput.Listeners {
			if *listener.Port == 80 && listener.Protocol == elbv2types.ProtocolEnumHttp {
				fmt.Printf("Listener already exists on port 80, updating target group\n")
				// Update the existing listener to point to the new target group
				_, updateErr := d.elbv2Client.ModifyListener(d.ctx, &elasticloadbalancingv2.ModifyListenerInput{
					ListenerArn: listener.ListenerArn,
					DefaultActions: []elbv2types.Action{
						{
							Type:           elbv2types.ActionTypeEnumForward,
							TargetGroupArn: aws.String(targetGroupArn),
						},
					},
				})
				if updateErr != nil {
					fmt.Printf("Warning: Failed to update existing listener: %v\n", updateErr)
				} else {
					fmt.Printf("Listener updated successfully\n")
					return nil
				}
			}
		}
	}

	// Create new listener
	input := &elasticloadbalancingv2.CreateListenerInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
		Protocol:        elbv2types.ProtocolEnumHttp,
		Port:            aws.Int32(80),
		DefaultActions: []elbv2types.Action{
			{
				Type:           elbv2types.ActionTypeEnumForward,
				TargetGroupArn: aws.String(targetGroupArn),
			},
		},
	}

	_, err = d.elbv2Client.CreateListener(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	fmt.Printf("Listener created successfully\n")
	return nil
}


func (d *ECSDeployer) Cleanup(config ECSConfig) error {
	fmt.Printf("Starting cleanup of ECS resources for service: %s\n", config.ServiceName)

	// Delete ECS service first
	err := d.deleteService(config.ClusterName, config.ServiceName)
	if err != nil {
		fmt.Printf("Warning: Failed to delete service: %v\n", err)
	}

	// Delete task definition
	err = d.deleteTaskDefinition(config.TaskDefinitionName)
	if err != nil {
		fmt.Printf("Warning: Failed to delete task definition: %v\n", err)
	}

	// Delete load balancer and associated resources
	err = d.deleteLoadBalancerResources(config.ServiceName)
	if err != nil {
		fmt.Printf("Warning: Failed to delete load balancer resources: %v\n", err)
	}

	// Delete cluster (if empty)
	err = d.deleteCluster(config.ClusterName)
	if err != nil {
		fmt.Printf("Warning: Failed to delete cluster: %v\n", err)
	}

	// Delete log groups
	err = d.deleteLogGroups(config.TaskDefinitionName)
	if err != nil {
		fmt.Printf("Warning: Failed to delete log groups: %v\n", err)
	}

	// Delete secrets if they were created
	if config.CreateSecrets {
		err = d.deleteSecrets(config.ServiceName)
		if err != nil {
			fmt.Printf("Warning: Failed to delete secrets: %v\n", err)
		}
	}

	// Delete EFS if it was created
	if config.CreateEFS && config.EFSVolumeId != "" {
		err = d.deleteEFS(config.EFSVolumeId, config.SubnetIds)
		if err != nil {
			fmt.Printf("Warning: Failed to delete EFS: %v\n", err)
		}
	}

	fmt.Printf("Cleanup completed for service: %s\n", config.ServiceName)
	return nil
}

func (d *ECSDeployer) deleteService(clusterName, serviceName string) error {
	fmt.Printf("Deleting ECS service: %s\n", serviceName)

	// First, update service to have 0 desired count
	_, err := d.ecsClient.UpdateService(d.ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(clusterName),
		Service:      aws.String(serviceName),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		return fmt.Errorf("failed to scale down service: %w", err)
	}

	// Wait for service to scale down
	fmt.Printf("Waiting for service %s to scale down...\n", serviceName)
	waiter := ecs.NewServicesStableWaiter(d.ecsClient)
	err = waiter.Wait(d.ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}, 5*time.Minute)
	if err != nil {
		fmt.Printf("Warning: Timeout waiting for service to scale down: %v\n", err)
	}

	// Delete the service
	_, err = d.ecsClient.DeleteService(d.ctx, &ecs.DeleteServiceInput{
		Cluster: aws.String(clusterName),
		Service: aws.String(serviceName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	fmt.Printf("ECS service %s deleted successfully\n", serviceName)
	return nil
}

func (d *ECSDeployer) deleteTaskDefinition(taskDefinitionName string) error {
	fmt.Printf("Deregistering task definition: %s\n", taskDefinitionName)

	// List all revisions of the task definition
	listOutput, err := d.ecsClient.ListTaskDefinitions(d.ctx, &ecs.ListTaskDefinitionsInput{
		FamilyPrefix: aws.String(taskDefinitionName),
		Status:       types.TaskDefinitionStatusActive,
	})
	if err != nil {
		return fmt.Errorf("failed to list task definitions: %w", err)
	}

	// Deregister all revisions
	for _, taskDefArn := range listOutput.TaskDefinitionArns {
		_, err := d.ecsClient.DeregisterTaskDefinition(d.ctx, &ecs.DeregisterTaskDefinitionInput{
			TaskDefinition: aws.String(taskDefArn),
		})
		if err != nil {
			fmt.Printf("Warning: Failed to deregister task definition %s: %v\n", taskDefArn, err)
		} else {
			fmt.Printf("Task definition %s deregistered\n", taskDefArn)
		}
	}

	return nil
}

func (d *ECSDeployer) deleteLoadBalancerResources(serviceName string) error {
	fmt.Printf("Deleting load balancer resources for service: %s\n", serviceName)

	loadBalancerName := fmt.Sprintf("%s-alb", serviceName)
	targetGroupName := fmt.Sprintf("%s-tg", serviceName)

	// Get load balancer ARN
	lbOutput, err := d.elbv2Client.DescribeLoadBalancers(d.ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		Names: []string{loadBalancerName},
	})
	if err != nil {
		fmt.Printf("Load balancer %s not found, skipping deletion\n", loadBalancerName)
		return nil
	}

	if len(lbOutput.LoadBalancers) == 0 {
		fmt.Printf("Load balancer %s not found\n", loadBalancerName)
		return nil
	}

	loadBalancerArn := *lbOutput.LoadBalancers[0].LoadBalancerArn

	// Delete listeners first
	listenersOutput, err := d.elbv2Client.DescribeListeners(d.ctx, &elasticloadbalancingv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
	})
	if err == nil {
		for _, listener := range listenersOutput.Listeners {
			_, err := d.elbv2Client.DeleteListener(d.ctx, &elasticloadbalancingv2.DeleteListenerInput{
				ListenerArn: listener.ListenerArn,
			})
			if err != nil {
				fmt.Printf("Warning: Failed to delete listener: %v\n", err)
			} else {
				fmt.Printf("Listener deleted: %s\n", *listener.ListenerArn)
			}
		}
	}

	// Delete load balancer
	_, err = d.elbv2Client.DeleteLoadBalancer(d.ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
	})
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}
	fmt.Printf("Load balancer %s deleted\n", loadBalancerName)

	// Wait a bit for load balancer to be deleted before deleting target group
	fmt.Printf("Waiting for load balancer to be deleted...\n")
	time.Sleep(30 * time.Second)

	// Delete target group
	tgOutput, err := d.elbv2Client.DescribeTargetGroups(d.ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
		Names: []string{targetGroupName},
	})
	if err != nil {
		fmt.Printf("Target group %s not found, skipping deletion\n", targetGroupName)
		return nil
	}

	if len(tgOutput.TargetGroups) > 0 {
		targetGroupArn := *tgOutput.TargetGroups[0].TargetGroupArn
		_, err = d.elbv2Client.DeleteTargetGroup(d.ctx, &elasticloadbalancingv2.DeleteTargetGroupInput{
			TargetGroupArn: aws.String(targetGroupArn),
		})
		if err != nil {
			return fmt.Errorf("failed to delete target group: %w", err)
		}
		fmt.Printf("Target group %s deleted\n", targetGroupName)
	}

	return nil
}

func (d *ECSDeployer) deleteCluster(clusterName string) error {
	fmt.Printf("Checking if cluster %s can be deleted\n", clusterName)

	// Check if cluster has any services
	servicesOutput, err := d.ecsClient.ListServices(d.ctx, &ecs.ListServicesInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to list services in cluster: %w", err)
	}

	if len(servicesOutput.ServiceArns) > 0 {
		fmt.Printf("Cluster %s still has %d services, not deleting\n", clusterName, len(servicesOutput.ServiceArns))
		return nil
	}

	// Check if cluster has any tasks
	tasksOutput, err := d.ecsClient.ListTasks(d.ctx, &ecs.ListTasksInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks in cluster: %w", err)
	}

	if len(tasksOutput.TaskArns) > 0 {
		fmt.Printf("Cluster %s still has %d tasks, not deleting\n", clusterName, len(tasksOutput.TaskArns))
		return nil
	}

	// Delete cluster
	_, err = d.ecsClient.DeleteCluster(d.ctx, &ecs.DeleteClusterInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	fmt.Printf("ECS cluster %s deleted successfully\n", clusterName)
	return nil
}

func (d *ECSDeployer) deleteLogGroups(taskDefinitionName string) error {
	fmt.Printf("Deleting log groups for task definition: %s\n", taskDefinitionName)

	webAppLogGroup := fmt.Sprintf("/ecs/%s-webapp", taskDefinitionName)
	dbLogGroup := fmt.Sprintf("/ecs/%s-database", taskDefinitionName)

	// Delete web app log group
	_, err := d.logsClient.DeleteLogGroup(d.ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: aws.String(webAppLogGroup),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to delete log group %s: %v\n", webAppLogGroup, err)
	} else {
		fmt.Printf("Log group %s deleted\n", webAppLogGroup)
	}

	// Delete database log group
	_, err = d.logsClient.DeleteLogGroup(d.ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: aws.String(dbLogGroup),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to delete log group %s: %v\n", dbLogGroup, err)
	} else {
		fmt.Printf("Log group %s deleted\n", dbLogGroup)
	}

	return nil
}

func (d *ECSDeployer) CreateSecrets(serviceName string) (map[string]string, error) {
	fmt.Printf("Creating secrets for service: %s\n", serviceName)

	secrets := make(map[string]string)

	// 1. Database secret
	dbSecret, err := d.generateRandomPassword(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate database password: %w", err)
	}
	dbSecretArn, err := d.createSecret(fmt.Sprintf("%s-db-password", serviceName), dbSecret, "Database password for Neo4j")
	if err != nil {
		return nil, fmt.Errorf("failed to create database secret: %w", err)
	}
	secrets["DB_SECRET_ARN"] = dbSecretArn

	// 2. JWT Secret
	jwtSecret, err := d.generateRandomPassword(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	jwtSecretArn, err := d.createSecret(fmt.Sprintf("%s-jwt-secret", serviceName), jwtSecret, "JWT secret for authentication")
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT secret: %w", err)
	}
	secrets["JWT_SECRET_ARN"] = jwtSecretArn

	// 3. Session Key
	sessionKey, err := d.generateRandomPassword(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}
	sessionKeyArn, err := d.createSecret(fmt.Sprintf("%s-session-key", serviceName), sessionKey, "Session key for session management")
	if err != nil {
		return nil, fmt.Errorf("failed to create session key: %w", err)
	}
	secrets["SESSION_KEY_ARN"] = sessionKeyArn

	// 4. Anthropic API Key (from environment variable)
	if anthropicKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicKey != "" {
		anthropicArn, err := d.createSecret(fmt.Sprintf("%s-anthropic-key", serviceName), anthropicKey, "Anthropic API key")
		if err != nil {
			return nil, fmt.Errorf("failed to create Anthropic API secret: %w", err)
		}
		secrets["ANTHROPIC_SECRET_ARN"] = anthropicArn
	}

	// 5. Gmail User (from environment variable)
	if gmailUser := os.Getenv("GMAIL_USER"); gmailUser != "" {
		gmailUserArn, err := d.createSecret(fmt.Sprintf("%s-gmail-user", serviceName), gmailUser, "Gmail user for email integration")
		if err != nil {
			return nil, fmt.Errorf("failed to create Gmail user secret: %w", err)
		}
		secrets["GMAIL_USER_ARN"] = gmailUserArn
	}

	// 6. Gmail Password (from environment variable)
	if gmailPass := os.Getenv("GMAIL_PASS"); gmailPass != "" {
		gmailPassArn, err := d.createSecret(fmt.Sprintf("%s-gmail-pass", serviceName), gmailPass, "Gmail password for email integration")
		if err != nil {
			return nil, fmt.Errorf("failed to create Gmail password secret: %w", err)
		}
		secrets["GMAIL_PASS_ARN"] = gmailPassArn
	}

	fmt.Printf("Created %d secrets successfully\n", len(secrets))
	return secrets, nil
}

func (d *ECSDeployer) createSecret(name, value, description string) (string, error) {
	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
		Description:  aws.String(description),
	}

	result, err := d.secretsClient.CreateSecret(d.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create secret %s: %w", name, err)
	}

	fmt.Printf("Created secret: %s\n", name)
	return *result.ARN, nil
}

func (d *ECSDeployer) generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range b {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		b[i] = charset[idx.Int64()]
	}
	return string(b), nil
}

func (d *ECSDeployer) CreateEFS(serviceName string, subnetIds []string, securityGroupIds []string) (string, error) {
	fmt.Printf("Creating EFS file system for service: %s\n", serviceName)

	// Create EFS file system
	createInput := &efs.CreateFileSystemInput{
		CreationToken:                aws.String(fmt.Sprintf("%s-efs", serviceName)),
		PerformanceMode:              efstypes.PerformanceModeGeneralPurpose,
		ThroughputMode:               efstypes.ThroughputModeProvisioned,
		ProvisionedThroughputInMibps: aws.Float64(10), // 10 MiB/s
		Tags: []efstypes.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(fmt.Sprintf("%s-neo4j-data", serviceName)),
			},
			{
				Key:   aws.String("Service"),
				Value: aws.String(serviceName),
			},
		},
	}

	result, err := d.efsClient.CreateFileSystem(d.ctx, createInput)
	if err != nil {
		return "", fmt.Errorf("failed to create EFS file system: %w", err)
	}

	efsId := *result.FileSystemId
	fmt.Printf("Created EFS file system: %s\n", efsId)

	// Wait for EFS to be available
	fmt.Printf("Waiting for EFS file system to be available...\n")
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		descOutput, err := d.efsClient.DescribeFileSystems(d.ctx, &efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(efsId),
		})
		if err != nil {
			return "", fmt.Errorf("failed to describe EFS: %w", err)
		}

		if len(descOutput.FileSystems) > 0 && descOutput.FileSystems[0].LifeCycleState == efstypes.LifeCycleStateAvailable {
			break
		}

		if i == 59 {
			return "", fmt.Errorf("timeout waiting for EFS to be available")
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("EFS file system %s is now available\n", efsId)

	// Create mount targets in all subnets
	if err := d.createEFSMountTargets(efsId, subnetIds, securityGroupIds); err != nil {
		return "", fmt.Errorf("failed to create EFS mount targets: %w", err)
	}

	return efsId, nil
}

func (d *ECSDeployer) createEFSMountTargets(efsId string, subnetIds []string, securityGroupIds []string) error {
	fmt.Printf("Creating EFS mount targets for file system: %s\n", efsId)

	for _, subnetId := range subnetIds {
		input := &efs.CreateMountTargetInput{
			FileSystemId:   aws.String(efsId),
			SubnetId:       aws.String(subnetId),
			SecurityGroups: securityGroupIds,
		}

		_, err := d.efsClient.CreateMountTarget(d.ctx, input)
		if err != nil {
			fmt.Printf("Warning: Failed to create mount target in subnet %s: %v\n", subnetId, err)
		} else {
			fmt.Printf("Created EFS mount target in subnet: %s\n", subnetId)
		}
	}

	return nil
}

func (d *ECSDeployer) deleteSecrets(serviceName string) error {
	fmt.Printf("Deleting secrets for service: %s\n", serviceName)

	secretNames := []string{
		fmt.Sprintf("%s-db-password", serviceName),
		fmt.Sprintf("%s-jwt-secret", serviceName),
		fmt.Sprintf("%s-session-key", serviceName),
		fmt.Sprintf("%s-anthropic-key", serviceName),
		fmt.Sprintf("%s-gmail-user", serviceName),
		fmt.Sprintf("%s-gmail-pass", serviceName),
	}

	for _, secretName := range secretNames {
		_, err := d.secretsClient.DeleteSecret(d.ctx, &secretsmanager.DeleteSecretInput{
			SecretId:                   aws.String(secretName),
			ForceDeleteWithoutRecovery: aws.Bool(true), // Immediate deletion without recovery period
		})
		if err != nil {
			fmt.Printf("Warning: Failed to delete secret %s: %v\n", secretName, err)
		} else {
			fmt.Printf("Deleted secret: %s\n", secretName)
		}
	}

	return nil
}

func (d *ECSDeployer) deleteEFS(efsId string, _ []string) error {
	fmt.Printf("Deleting EFS file system: %s\n", efsId)

	// First delete all mount targets
	mountTargetsOutput, err := d.efsClient.DescribeMountTargets(d.ctx, &efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(efsId),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to describe mount targets: %v\n", err)
	} else {
		for _, mountTarget := range mountTargetsOutput.MountTargets {
			_, err := d.efsClient.DeleteMountTarget(d.ctx, &efs.DeleteMountTargetInput{
				MountTargetId: mountTarget.MountTargetId,
			})
			if err != nil {
				fmt.Printf("Warning: Failed to delete mount target %s: %v\n", *mountTarget.MountTargetId, err)
			} else {
				fmt.Printf("Deleted EFS mount target: %s\n", *mountTarget.MountTargetId)
			}
		}

		// Wait for mount targets to be deleted
		if len(mountTargetsOutput.MountTargets) > 0 {
			fmt.Printf("Waiting for mount targets to be deleted...\n")
			time.Sleep(30 * time.Second)
		}
	}

	// Delete the file system
	_, err = d.efsClient.DeleteFileSystem(d.ctx, &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(efsId),
	})
	if err != nil {
		return fmt.Errorf("failed to delete EFS file system: %w", err)
	}

	fmt.Printf("EFS file system %s deleted successfully\n", efsId)
	return nil
}
