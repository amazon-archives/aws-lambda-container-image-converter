// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package types

import (
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/clients"
)

type LambdaLayer struct {
	Digest string
	File   string
}

type CmdOptions struct {
	Image          string // Name of the container image
	Region         string // AWS region
	OutputDir      string // Output directory for the Lambda layers
	DryRun         bool   // Dry-run (will not register with Lambda)
	LayerNamespace string // Prefix for published Lambda layers
	Description    string // Description of the current layer version
	LicenseInfo    string // Layer's software license
}

type PublishOptions struct {
	LambdaClient    lambdaiface.LambdaAPI
	LayerPrefix     string
	ResultsDir      string
	SourceImageName string
	Description     string
	LicenseInfo     string
}

func ConvertToPublishOptions(opts *CmdOptions) *PublishOptions {
	return &PublishOptions{
		SourceImageName: opts.Image,
		LambdaClient:    clients.NewLambdaClient(opts.Region),
		LayerPrefix:     opts.LayerNamespace,
		ResultsDir:      opts.OutputDir,
		Description:     opts.Description,
		LicenseInfo:     opts.LicenseInfo,
	}
}
