#!/bin/bash

set -e

PROJECT_ID=${1:-"gen-lang-client-0715048106"}
REGION=${2:-"asia-northeast1"}
DEPLOYMENT_NAME="main"

echo "🚀 Deploying with Deployment Manager"
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo ""

# Check required environment variables
if [[ -z "$GEMINI_API_KEY" || -z "$SLACK_BOT_TOKEN" || -z "$WEBHOOK_AUTH_TOKEN" ]]; then
    echo "❌ Required environment variables missing:"
    echo "   GEMINI_API_KEY, SLACK_BOT_TOKEN, WEBHOOK_AUTH_TOKEN"
    exit 1
fi

# Set the project
gcloud config set project $PROJECT_ID

# Create source code archive
echo "📦 Creating source code archive..."
cd /Users/pepe/ghq/github.com/pep299/article-summarizer-v3/app
zip -r ../infra/gcp/function-source.zip . -x "*.git*" "*.DS_Store*" "coverage*" "test/*"
cd ../infra/gcp

# Upload source to Cloud Storage
echo "☁️ Uploading source code..."
BUCKET_NAME="${PROJECT_ID}-function-source"
gsutil mb -p $PROJECT_ID -l $REGION gs://$BUCKET_NAME/ 2>/dev/null || echo "Bucket already exists"
gsutil cp function-source.zip gs://$BUCKET_NAME/$DEPLOYMENT_NAME.zip

# Deploy using Deployment Manager
echo "🚀 Deploying infrastructure..."
gcloud deployment-manager deployments create $DEPLOYMENT_NAME \
    --config deployment.yaml \
    --properties geminiApiKey="$GEMINI_API_KEY",slackBotToken="$SLACK_BOT_TOKEN",webhookAuthToken="$WEBHOOK_AUTH_TOKEN" \
    --project $PROJECT_ID

echo ""
echo "✅ Deployment completed successfully!"
echo "Function URL: $(gcloud functions describe article-summarizer-v3 --region=$REGION --format="value(httpsTrigger.url)" 2>/dev/null || echo 'Deploying...')"

# Cleanup
rm -f function-source.zip