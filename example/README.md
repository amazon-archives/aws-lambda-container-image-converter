Example for creating an custom Lambda Layer to run PHP 7
========================================================

## Quickstart

```bash

## Build the docker image

docker build -t lambda-php .

## Test locally using Docker
## The bootstrap script will be the entrypoint,
##   which will then run the specified lambda function,
##   and pass the input data to the lambda function.

docker run lambda-php hello '{"name": "World"}'

## Install img2lambda

wget -O /usr/local/bin/img2lambda \
     https://github.com/awslabs/aws-lambda-container-image-converter/releases/download/0.3.0/linux-amd64-img2lambda
chmod 0755 /usr/local/bin/img2lambda

## Deploy lambda layer

img2lambda -i lambda-php:latest -r us-east-1 -n lambda-php

## Deploy Lambda functions: hello and goodbye
## img2lambda creates `output/layers.yaml` which will be combined with `template.yaml` to create `template-with-layers.yaml`

export AWS_REGION="us-east-1"
export MY_LAMBDA_LAYER_BUCKET="devokun-lambda-php"


sed -i 's/^- /      - /' ../output/layers.yaml && \
sed -e "/LAYERS_PLACEHOLDER/r ../output/layers.yaml" \
    -e "s///" template.yaml > template-with-layers.yaml


## Install AWS CLI

pip3 install aws-cli

## Create S3 bucket for storing Lambda Functions

aws s3 mb s3://${MY_LAMBDA_LAYER_BUCKET}

## Install SAM CLI

pip3 install aws-sam-cli

## Deploy Lambda functions using SAM

sam package --template-file template-with-layers.yaml \
            --output-template-file packaged.yaml \
            --region ${AWS_REGION} \
            --s3-bucket ${MY_LAMBDA_LAYER_BUCKET}

sam deploy --template-file packaged.yaml \
           --capabilities CAPABILITY_IAM  \
           --region ${AWS_REGION} \
           --stack-name img2lambda-php-example

```

