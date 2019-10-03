#!/bin/bash

. ../demo-magic/demo-magic.sh

# Clean up the account

AWS_ACCOUNT_ID=`aws sts get-caller-identity --query 'Account' --output text`

aws lambda delete-function --region us-east-1 --function-name php-example-hello

for layerName in $(aws lambda list-layers --region us-east-1 | jq -r '.Layers[] | select(.LayerName | contains("img2lambda")) | .LayerName'); do
    for version in $(aws lambda list-layer-versions --region us-east-1 --layer-name "$layerName" | jq -r '.LayerVersions[].Version'); do
        aws lambda delete-layer-version --region us-east-1 --layer-name "$layerName" --version-number "$version" 1>&2
    done
done

cd example/

clear

# Look and feel

TYPE_SPEED=15
DEMO_COMMENT_COLOR=$CYAN
NO_WAIT=false

# Start the demo

PROMPT_TIMEOUT=0

p "# Welcome to img2lambda!"

PROMPT_TIMEOUT=1

p "# Let's create a custom PHP runtime and a PHP function for AWS Lambda using Docker"

pe "less Dockerfile"

pe "docker build -t lambda-php ."

p "# Love that fast Docker build caching!"

p "# Let's test our PHP runtime and function locally with Docker"

pe "docker run lambda-php hello '{\"name\": \"World\"}'"

p "# img2lambda will extract our PHP function from the Docker image, and zip the files into a Lambda deployment package"

p "# img2lambda will also extract our PHP runtime from the Docker image as Lambda layers, and publish the layers to Lambda"

pe "img2lambda -i lambda-php:latest -r us-east-1 -o ./output"

p "# img2lambda found our PHP function files in the Docker image and zipped them up"

pe "unzip -l output/function.zip"

p "# img2lambda also found 2 layers in the Docker image that contain our PHP runtime files, so it created 2 Lambda layers and published them"

pe "more output/layers.json"

p "# Now we can create a Lambda function that uses the deployment package and the published layers"

TYPE_SPEED=''
pe "aws lambda create-function --function-name php-example-hello --zip-file fileb://./output/function.zip --layers file://./output/layers.json --runtime provided --handler hello --role \"arn:aws:iam::$AWS_ACCOUNT_ID:role/service-role/LambdaPhpExample\" --region us-east-1"
TYPE_SPEED=15

p "# Let's now invoke the function and test out our PHP custom runtime"

TYPE_SPEED=''
pe "aws lambda invoke --function-name php-example-hello --payload '{\"name\": \"World\"}' --region us-east-1 --log-type Tail --query 'LogResult' --output text hello-output.txt | base64 --decode"
TYPE_SPEED=15

pe "more hello-output.txt"

p "# Enjoy building your Lambda functions and layers with Docker and img2lambda!"
