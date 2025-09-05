#!/bin/bash

# Script to create IAM user with least privilege for OpsAgents application
# Run this script with admin AWS credentials

set -e

USER_NAME="opsagents-app-user"
POLICY_NAME="OpsAgentsLightsailPolicy"
POLICY_FILE="iam-policy.json"

echo "Creating IAM user: $USER_NAME"

# Create IAM user
aws iam create-user \
    --user-name "$USER_NAME" \
    --tags "Key=Application,Value=OpsAgents" "Key=Purpose,Value=LightsailDeployment" \
    --path "/"

# Create IAM policy
aws iam create-policy \
    --policy-name "$POLICY_NAME" \
    --policy-document "file://$POLICY_FILE" \
    --description "Least privilege policy for OpsAgents Lightsail deployment"

# Get account ID
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Attach policy to user
aws iam attach-user-policy \
    --user-name "$USER_NAME" \
    --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/$POLICY_NAME"

# Create access key
echo "Creating access key for user..."
KEY_OUTPUT=$(aws iam create-access-key --user-name "$USER_NAME")

echo "IAM user created successfully!"
echo "User Name: $USER_NAME"
echo "Policy: $POLICY_NAME"
echo ""
echo "Access Key ID: $(echo "$KEY_OUTPUT" | jq -r '.AccessKey.AccessKeyId')"
echo "Secret Access Key: $(echo "$KEY_OUTPUT" | jq -r '.AccessKey.SecretAccessKey')"
echo ""
echo "IMPORTANT: Save these credentials securely. The secret key will not be shown again."
echo "Update your launch.json file with these new credentials."