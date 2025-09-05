#!/bin/bash

# Script to create IAM resources for OpsAgents application
# Run this script with AWS credentials that have IAM administrative permissions

set -e

USER_NAME="opsagents-app-user"
POLICY_NAME="OpsAgentsLightsailPolicy"

echo "🔐 Creating IAM resources for OpsAgents application..."
echo "User: $USER_NAME"
echo "Policy: $POLICY_NAME"
echo ""

# Check if user already exists
if aws iam get-user --user-name "$USER_NAME" &>/dev/null; then
    echo "⚠️  User $USER_NAME already exists. Skipping user creation."
else
    echo "👤 Creating IAM user: $USER_NAME"
    aws iam create-user \
        --user-name "$USER_NAME" \
        --tags "Key=Application,Value=OpsAgents" "Key=Purpose,Value=LightsailDeployment" "Key=CreatedBy,Value=OpsAgentsSetup" \
        --path "/"
    echo "✅ User created successfully"
fi

# Get account ID for policy ARN
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/$POLICY_NAME"

# Check if policy already exists
if aws iam get-policy --policy-arn "$POLICY_ARN" &>/dev/null; then
    echo "⚠️  Policy $POLICY_NAME already exists. Skipping policy creation."
else
    echo "📋 Creating IAM policy: $POLICY_NAME"
    
    # Create the policy document
    cat > /tmp/policy.json << 'EOF'
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "LightsailContainerServiceAccess",
            "Effect": "Allow",
            "Action": [
                "lightsail:CreateContainerService",
                "lightsail:CreateContainerServiceDeployment",
                "lightsail:GetContainerServices",
                "lightsail:GetContainerService",
                "lightsail:UpdateContainerService",
                "lightsail:DeleteContainerService",
                "lightsail:GetContainerServiceDeployments"
            ],
            "Resource": "*"
        },
        {
            "Sid": "LightsailRegionAccess",
            "Effect": "Allow", 
            "Action": [
                "lightsail:GetRegions",
                "lightsail:GetAvailabilityZones"
            ],
            "Resource": "*"
        }
    ]
}
EOF

    # Create IAM policy
    aws iam create-policy \
        --policy-name "$POLICY_NAME" \
        --policy-document "file:///tmp/policy.json" \
        --description "Least privilege policy for OpsAgents Lightsail deployment" \
        --tags "Key=Application,Value=OpsAgents" "Key=Purpose,Value=LightsailDeployment" "Key=CreatedBy,Value=OpsAgentsSetup"
    
    echo "✅ Policy created successfully"
fi

# Check if policy is already attached to user
if aws iam list-attached-user-policies --user-name "$USER_NAME" --query "AttachedPolicies[?PolicyArn=='$POLICY_ARN']" --output text | grep -q "$POLICY_ARN"; then
    echo "⚠️  Policy already attached to user. Skipping policy attachment."
else
    echo "🔗 Attaching policy to user..."
    aws iam attach-user-policy \
        --user-name "$USER_NAME" \
        --policy-arn "$POLICY_ARN"
    echo "✅ Policy attached successfully"
fi

# Check if user already has access keys
EXISTING_KEYS=$(aws iam list-access-keys --user-name "$USER_NAME" --query 'AccessKeyMetadata[].AccessKeyId' --output text)

if [ -n "$EXISTING_KEYS" ]; then
    echo "⚠️  User already has access keys:"
    echo "$EXISTING_KEYS"
    echo ""
    echo "If you need new keys, delete the old ones first:"
    for key in $EXISTING_KEYS; do
        echo "aws iam delete-access-key --user-name $USER_NAME --access-key-id $key"
    done
else
    echo "🔑 Creating access key for user..."
    KEY_OUTPUT=$(aws iam create-access-key --user-name "$USER_NAME")
    
    ACCESS_KEY_ID=$(echo "$KEY_OUTPUT" | jq -r '.AccessKey.AccessKeyId')
    SECRET_ACCESS_KEY=$(echo "$KEY_OUTPUT" | jq -r '.AccessKey.SecretAccessKey')
    
    echo ""
    echo "🎉 IAM resources created successfully!"
    echo "=================================="
    echo "User Name: $USER_NAME"
    echo "Policy ARN: $POLICY_ARN"
    echo ""
    echo "🔐 CREDENTIALS (save these securely):"
    echo "AWS_ACCESS_KEY_ID=$ACCESS_KEY_ID"
    echo "AWS_SECRET_ACCESS_KEY=$SECRET_ACCESS_KEY"
    echo ""
    echo "⚠️  IMPORTANT: Save these credentials securely!"
    echo "The secret key will not be shown again."
    echo ""
    echo "Next steps:"
    echo "1. Copy these credentials to your .env file"
    echo "2. Update your VS Code launch configuration"
    echo "3. Test your application with the new credentials"
fi

# Clean up temporary file
rm -f /tmp/policy.json

echo ""
echo "✅ Setup complete!"