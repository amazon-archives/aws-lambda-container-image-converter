// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package types

import (
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/clients"
)

type LambdaDeploymentPackage struct {
	FileCount int
	File      string
}

type LambdaLayer struct {
	Digest string
	File   string
}

type CmdOptions struct {
	Image              string   // Name of the container image
	ImageType          string   // Type of the container image
	Region             string   // AWS region
	Profile            string   // AWS credentials profile
	OutputDir          string   // Output directory for the Lambda layers
	DryRun             bool     // Dry-run (will not register with Lambda)
	LayerNamespace     string   // Prefix for published Lambda layers
	Description        string   // Description of the current layer version
	LicenseInfo        string   // Layer's software license
	CompatibleRuntimes []string // A list of function runtimes compatible with the current layer
}

type PublishOptions struct {
	LambdaClient       lambdaiface.LambdaAPI
	LayerPrefix        string
	ResultsDir         string
	SourceImageName    string
	Description        string
	LicenseInfo        string
	CompatibleRuntimes []string
}

func ConvertToPublishOptions(opts *CmdOptions) *PublishOptions {
	return &PublishOptions{
		SourceImageName:    opts.Image,
		LambdaClient:       clients.NewLambdaClient(opts.Region, opts.Profile),
		LayerPrefix:        opts.LayerNamespace,
		ResultsDir:         opts.OutputDir,
		Description:        opts.Description,
		LicenseInfo:        opts.LicenseInfo,
		CompatibleRuntimes: opts.CompatibleRuntimes,
	}
}

// valid aws lambda function runtimes
type Runtimes []string

// utility function to validate if a runtime is valid (supported by aws) or not
func (r Runtimes) Contains(runtime string) bool {
	for _, value := range r {
		if value == runtime {
			return true
		}
	}
	return false
}

// a list of aws supported runtimes as of 2020-06-18
var ValidRuntimes = Runtimes{
	// eol'ed runtimes are included to support existing functions
	"nodejs",    // eol
	"nodejs4.3", // eol
	"nodejs4.3-edge", // eol
	"nodejs6.10", // eol
	"nodejs8.10", // eol
	"nodejs10.x",
	"nodejs12.x",
	"java8",
	"java11",
	"python2.7",
	"python3.6",
	"python3.7",
	"python3.8",
	"dotnetcore1.0", // eol
	"dotnetcore2.0", // eol
	"dotnetcore2.1",
	"dotnetcore3.1",
	"go1.x",
	"ruby2.5",
	"ruby2.7",
	"provided",
}
