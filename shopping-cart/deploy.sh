#!/bin/bash
# Deploy Shopping Cart - builds backend + frontend and deploys to AWS
set -euo pipefail

STACK_NAME="shopping-cart"
REGION="${AWS_REGION:-us-east-1}"

echo "========================================="
echo "  Shopping Cart - Deploy"
echo "========================================="
echo ""

# Step 1: Build and deploy SAM stack (Lambda + API Gateway + DynamoDB)
echo ">> [1/4] Building backend..."
sam build --template-file template.yaml

echo ">> [2/4] Deploying backend to AWS..."
PARAM_OVERRIDES=""
if [[ -n "${FRONTEND_BUCKET_NAME:-}" ]]; then
  PARAM_OVERRIDES="--parameter-overrides FrontendBucketName=${FRONTEND_BUCKET_NAME}"
fi

sam deploy \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --resolve-s3 \
  --capabilities CAPABILITY_IAM \
  --no-confirm-changeset \
  --no-fail-on-empty-changeset \
  $PARAM_OVERRIDES

# Step 2: Get stack outputs
echo ">> [3/4] Fetching deployment outputs..."
API_URL=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiUrl`].OutputValue' \
  --output text)

FRONTEND_BUCKET=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`FrontendBucketName`].OutputValue' \
  --output text)

FRONTEND_URL=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`FrontendUrl`].OutputValue' \
  --output text)

# Step 3: Build and deploy frontend
echo ">> [4/4] Building and deploying frontend..."
cd frontend

# Set the API URL for the React build
export VITE_API_URL="$API_URL"

# Install dependencies and build
npm install --silent
npm run build

# Upload to S3
aws s3 sync dist/ "s3://${FRONTEND_BUCKET}/" \
  --region "$REGION" \
  --delete

cd ..

echo ""
echo "========================================="
echo "  Deployment Complete!"
echo "========================================="
echo ""
echo "  API URL:      ${API_URL}"
echo "  Frontend URL: ${FRONTEND_URL}"
echo ""
echo "  Open the Frontend URL in your browser."
echo "  After making changes, run ./deploy.sh again."
echo "========================================="
