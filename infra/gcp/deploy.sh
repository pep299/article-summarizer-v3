#!/bin/bash

set -e

PROJECT_ID=${1:-"article-summarizer-project"}
REGION=${2:-"us-central1"}
DEPLOYMENT_NAME="article-summarizer-v3"

echo "🚀 Deploying Article Summarizer v3 to Google Cloud"
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo ""

# Set the project
echo "📝 Setting project..."
gcloud config set project $PROJECT_ID

# Enable required APIs
echo "🔧 Enabling required APIs..."
gcloud services enable cloudfunctions.googleapis.com
gcloud services enable cloudscheduler.googleapis.com
gcloud services enable storage.googleapis.com
gcloud services enable deploymentmanager.googleapis.com

# Create source code archive
echo "📦 Creating source code archive..."
zip -r function-source.zip . -x "*.git*" "*.DS_Store*" "deploy.sh" "README.md" "*.yaml" "Dockerfile" "docker-compose.yml" "Makefile"

# Upload source to Cloud Storage
echo "☁️ Uploading source code..."
BUCKET_NAME="${PROJECT_ID}-function-source"
gsutil mb -p $PROJECT_ID -l $REGION gs://$BUCKET_NAME/ 2>/dev/null || echo "Bucket already exists"
gsutil cp function-source.zip gs://$BUCKET_NAME/$DEPLOYMENT_NAME.zip

# Deploy using Deployment Manager
echo "🚀 Deploying infrastructure..."
gcloud deployment-manager deployments create $DEPLOYMENT_NAME \
    --config deployment.yaml \
    --project $PROJECT_ID

echo ""
echo "✅ Deployment completed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Set up environment variables in Cloud Function:"
echo "   - GEMINI_API_KEY"
echo "   - SLACK_BOT_TOKEN"
echo "   - SLACK_CHANNEL"
echo ""
echo "2. Test the function:"
FUNCTION_URL=$(gcloud functions describe article-summarizer-v3 --region=$REGION --format="value(httpsTrigger.url)")
echo "   curl -X POST $FUNCTION_URL/api/v1/health"
echo ""
echo "3. Monitor logs:"
echo "   gcloud functions logs read article-summarizer-v3 --region=$REGION"

# Cleanup
rm -f function-source.zip