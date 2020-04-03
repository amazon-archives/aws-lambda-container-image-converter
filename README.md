## AWS Lambda Container Image Converter

[![Build Status](https://travis-ci.org/awslabs/aws-lambda-container-image-converter.svg?branch=master)](https://travis-ci.org/awslabs/aws-lambda-container-image-converter)

This container image converter tool (img2lambda) extracts an AWS Lambda function deployment package from a container image (such as a Docker image).
It also extracts AWS Lambda layers from a container image, and publishes them as new layer versions to Lambda.

![img2lambda Demo](assets/demo.gif)

To extract a Lambda function deployment package, the tool copies all files under '/var/task' in the container image into a deployment package zip file.

To extract Lambda layers, the tool copies all files under '/opt' in the container image, repackaging the individual container image layers as individual Lambda layer zip files.
The published layer ARNs are stored in a file 'output/layers.json', which can be used as input when creating Lambda functions.
Each layer is named using a "namespace" prefix (like 'img2lambda' or 'my-docker-image') and the SHA256 digest of the container image layer, in order to provide a way of tracking the provenance of the Lambda layer back to the container image that created it.
If a layer is already published to Lambda (same layer name, SHA256 digest, and size), it will not be published again.
Instead the existing layer version ARN will be written to the output file.

**Table of Contents**

<!-- toc -->

- [Usage](#usage)
- [Install](#install)
    + [Binaries](#binaries)
    + [From Source](#from-source)
- [Permissions](#permissions)
- [Examples](#examples)
    + [Docker Example](#docker-example)
    + [OCI Example](#oci-example)
    + [Deploy Manually](#deploy-manually)
    + [Deploy with AWS Serverless Application Model (SAM)](#deploy-with-aws-serverless-application-model-sam)
    + [Deploy with Serverless Framework](#deploy-with-serverless-framework)
- [License Summary](#license-summary)
- [Security Disclosures](#security-disclosures)

<!-- tocstop -->

## Usage

```
USAGE:
   img2lambda [options]

GLOBAL OPTIONS:
   --image value, -i value                 Name or path of the source container image. For example, 'my-docker-image:latest' or './my-oci-image-archive'. The image must be pulled locally already.
   --image-type value, -t value            Type of the source container image. Valid values: 'docker' (Docker image from the local Docker daemon), 'oci' (OCI image archive at the given path and optional tag) (default: "docker")
   --region value, -r value                AWS region (default: "us-east-1")
   --profile value, -p value               AWS credentials profile. Credentials will default to the same chain as the AWS CLI: environment variables, default profile, container credentials, EC2 instance credentials
   --output-directory value, -o value      Destination directory for output: function deployment package (function.zip) and list of published layers (layers.json, layers.yaml) (default: "./output")
   --layer-namespace value, -n value       Prefix for the layers published to Lambda (default: "img2lambda")
   --dry-run, -d                           Conduct a dry-run: Repackage the image, but only write the Lambda layers to local disk (do not publish to Lambda)
   --description value, --desc value       The description of this layer version (default: "created by img2lambda from image <name of the image>")
   --license-info value, -l value          The layer's software license. It can be an SPDX license identifier, the URL of the license hosted on the internet, or the full text of the license (default: no license)
   --compatible-runtime value, --cr value  An AWS Lambda function runtime compatible with the image layers. To specify multiple runtimes, repeat the option: --cr provided --cr python2.7 (default: "provided")
   --help, -h                              show help
   --version, -v                           print the version
```

Note: To avoid any sensitive information being packaged into a Lambda function deployment package or Lambda layer, do not store any sensitive information like credentials in the source container image's /opt or /var/task directories.

## Install

#### Binaries

Download pre-built binaries from the [Releases Page](https://github.com/awslabs/aws-lambda-container-image-converter/releases).

#### From Source

With go 1.11+:

```
$ git clone https://github.com/awslabs/aws-lambda-container-image-converter
$ cd aws-lambda-container-image-converter
$ make
$ ./bin/local/img2lambda --help
```

## Permissions

No credentials are required for dry-runs of the img2lambda tool.  When publishing layers to Lambda, img2lambda will look for credentials in the following order (using the default provider chain in the AWS SDK for Go).

1. Environment variables.
1. Shared credentials file.
1. If running on Amazon ECS (with task role) or AWS CodeBuild, IAM role from the container credentials endpoint.
1. If running on an Amazon EC2 instance, IAM role for Amazon EC2.

The credentials must have the following permissions:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "MinimalPermissions",
            "Effect": "Allow",
            "Action": [
                "lambda:GetLayerVersion",
                "lambda:ListLayerVersions",
                "lambda:PublishLayerVersion"
            ],
            "Resource": [
                "arn:aws:lambda:<REGION>:<ACCOUNT ID>:layer:<LAYER NAMESPACE>-sha256-*",
                "arn:aws:lambda:<REGION>:<ACCOUNT ID>:layer:<LAYER NAMESPACE>-sha256-*:*"
            ]
        }
    ]
}
```

For example:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "MinimalPermissions",
            "Effect": "Allow",
            "Action": [
                "lambda:GetLayerVersion",
                "lambda:ListLayerVersions",
                "lambda:PublishLayerVersion"
            ],
            "Resource": [
                "arn:aws:lambda:us-east-1:123456789012:layer:img2lambda-sha256-*",
                "arn:aws:lambda:us-east-1:123456789012:layer:img2lambda-sha256-*:*"
            ]
        }
    ]
}
```

## Examples

### Docker Example

Build the example Docker image to create a PHP Lambda custom runtime and Hello World PHP function:
```
cd example

docker build -t lambda-php .
```

The Hello World function can be invoked locally by running the Docker image:
```
docker run lambda-php hello '{"name": "World"}'

docker run lambda-php goodbye '{"name": "World"}'
```

Run the tool to both create a Lambda deployment package that contains the Hello World PHP function, and to create and publish Lambda layers that contain the PHP custom runtime:
```
../bin/local/img2lambda -i lambda-php:latest -r us-east-1 -o ./output
```

### OCI Example

Create an OCI image from the example Dockerfile:
```
cd example

podman build --format oci -t lambda-php .

podman push lambda-php oci-archive:./lambda-php-oci
```

Run the tool to both create a Lambda deployment package that contains the Hello World PHP function, and to create and publish Lambda layers that contain the PHP custom runtime:
```
../bin/local/img2lambda -i ./lambda-php-oci -t oci -r us-east-1 -o ./output
```

### Deploy Manually
Create a PHP function that uses the layers and deployment package extracted from the container image:
```
aws lambda create-function \
    --function-name php-example-hello \
    --handler hello \
    --zip-file fileb://./output/function.zip \
    --runtime provided \
    --role "arn:aws:iam::XXXXXXXXXXXX:role/service-role/LambdaPhpExample" \
    --region us-east-1 \
    --layers file://./output/layers.json
```

Finally, invoke the function:
```
aws lambda invoke \
    --function-name php-example-hello \
    --region us-east-1 \
    --log-type Tail \
    --query 'LogResult' \
    --output text \
    --payload '{"name": "World"}' hello-output.txt | base64 --decode

cat hello-output.txt
```

### Deploy with AWS Serverless Application Model (SAM)

See [the sample template.yaml](example/deploy/template.yaml) and [the sample template.json](example/deploy/template.json).

Insert the layers ARNs into the function definition:
```
cd example/deploy

sed -i 's/^- /      - /' ../output/layers.yaml && \
    sed -e "/LAYERS_PLACEHOLDER/r ../output/layers.yaml" -e "s///" template.yaml > template-with-layers.yaml

OR

cd example/deploy

sed -e "/\"LAYERS_PLACEHOLDER\"/r ../output/layers.json" -e "s///" template.json | jq . > template-with-layers.json
```

Deploy the function:
```
sam package --template-file template-with-layers.yaml \
            --output-template-file packaged.yaml \
            --region us-east-1 \
            --s3-bucket <bucket name>

sam deploy --template-file packaged.yaml \
           --capabilities CAPABILITY_IAM  \
           --region us-east-1 \
           --stack-name img2lambda-php-example

OR

sam package --template-file template-with-layers.json \
            --output-template-file packaged.json \
            --region us-east-1 \
            --s3-bucket <bucket name>

sam deploy --template-file packaged.json \
           --capabilities CAPABILITY_IAM  \
           --region us-east-1 \
           --stack-name img2lambda-php-example
```

Invoke the function:
```
aws lambda invoke \
    --function-name sam-php-example-hello \
    --region us-east-1 \
    --log-type Tail \
    --query 'LogResult' \
    --output text \
    --payload '{"name": "World"}' hello-output.txt | base64 --decode

cat hello-output.txt
```

### Deploy with Serverless Framework

See [the sample serverless.yml](example/deploy/serverless.yml) for how to use the img2lambda-generated layers in your Serverless function.

Deploy the function:
```
cd example/deploy

serverless deploy -v
```

Invoke the function:
```
serverless invoke -f hello -l -d '{"name": "World"}'
```

## License Summary

This sample code is made available under a modified MIT license. See the LICENSE file.

## Security Disclosures

If you would like to report a potential security issue in this project, please do not create a GitHub issue.  Instead, please follow the instructions [here](https://aws.amazon.com/security/vulnerability-reporting/) or [email AWS security directly](mailto:aws-security@amazon.com).
