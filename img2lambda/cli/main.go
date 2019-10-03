// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/extract"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/publish"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/version"
	"github.com/urfave/cli"
)

func createApp() (*cli.App, *types.CmdOptions) {
	opts := types.CmdOptions{}

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "img2lambda"
	app.Version = version.VersionString()
	app.Usage = "Repackages a container image into an AWS Lambda function deployment package. Extracts AWS Lambda layers from the image and publishes them to Lambda"
	app.Action = func(c *cli.Context) error {
		// parse and store the passed runtime list into the options object
		opts.CompatibleRuntimes = c.StringSlice("cr")

		validateCliOptions(&opts, c)
		return repackImageAction(&opts, c)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "image, i",
			Usage:       "Name or path of the source container image. For example, 'my-docker-image:latest' or './my-oci-image-archive'. The image must be pulled locally already.",
			Destination: &opts.Image,
		},
		cli.StringFlag{
			Name:        "image-type, t",
			Usage:       "Type of the source container image. Valid values: 'docker' (Docker image from the local Docker daemon), 'oci' (OCI image archive at the given path)",
			Value:       "docker",
			Destination: &opts.ImageType,
		},
		cli.StringFlag{
			Name:        "region, r",
			Usage:       "AWS region",
			Value:       "us-east-1",
			Destination: &opts.Region,
		},
		cli.StringFlag{
			Name:        "profile, p",
			Usage:       "AWS credentials profile. Credentials will default to the same chain as the AWS CLI: environment variables, default profile, container credentials, EC2 instance credentials",
			Destination: &opts.Profile,
		},
		cli.StringFlag{
			Name:        "output-directory, o",
			Usage:       "Destination directory for output: function deployment package (function.zip) and list of published layers (layers.json, layers.yaml)",
			Value:       "./output",
			Destination: &opts.OutputDir,
		},
		cli.StringFlag{
			Name:        "layer-namespace, n",
			Usage:       "Prefix for the layers published to Lambda",
			Value:       "img2lambda",
			Destination: &opts.LayerNamespace,
		},
		cli.BoolFlag{
			Name:        "dry-run, d",
			Usage:       "Conduct a dry-run: Repackage the image, but only write the Lambda layers to local disk (do not publish to Lambda)",
			Destination: &opts.DryRun,
		},
		cli.StringFlag{
			Name:        "description, desc",
			Usage:       "The description of this layer version (default: \"created by img2lambda from image <name of the image>\")",
			Destination: &opts.Description,
		},
		cli.StringFlag{
			Name:        "license-info, l",
			Usage:       "The layer's software license. It can be an SPDX license identifier, the URL of the license hosted on the internet, or the full text of the license (default: no license)",
			Destination: &opts.LicenseInfo,
		},
		cli.StringSliceFlag{
			Name:  "compatible-runtime, cr",
			Usage: "An AWS Lambda function runtime compatible with the image layers. To specify multiple runtimes, repeat the option: --cr provided --cr python2.7 (default: \"provided\")",
			Value: &cli.StringSlice{},
		},
	}

	app.Setup()
	app.Commands = []cli.Command{}

	return app, &opts
}

func validateCliOptions(opts *types.CmdOptions, context *cli.Context) {
	if opts.Image == "" {
		fmt.Print("ERROR: Image name is required\n\n")
		cli.ShowAppHelpAndExit(context, 1)
	}

	for _, runtime := range opts.CompatibleRuntimes {
		if !types.ValidRuntimes.Contains(runtime) {
			fmt.Println("ERROR: Compatible runtimes must be one of the supported runtimes\n\n", types.ValidRuntimes)
			cli.ShowAppHelpAndExit(context, 1)
		}
	}
}

func repackImageAction(opts *types.CmdOptions, context *cli.Context) error {
	var imageTransport string
	switch opts.ImageType {
	case "docker":
		imageTransport = "docker-daemon:"
	case "oci":
		imageTransport = "oci-archive:"
	default:
		fmt.Println("ERROR: Image type must be one of the supported image types")
		cli.ShowAppHelpAndExit(context, 1)
	}

	imageLocation := imageTransport + opts.Image
	if opts.ImageType == "docker" && strings.Count(imageLocation, ":") == 1 {
		imageLocation += ":latest"
	}

	layers, function, err := extract.RepackImage(imageLocation, opts.OutputDir)
	if err != nil {
		return err
	}

	if function.FileCount == 0 {
		// remove empty zip file
		os.Remove(function.File)
	}

	if len(layers) == 0 && function.FileCount == 0 {
		return errors.New("No compatible layers or function files found in the image (likely nothing found in /opt and /var/task)")
	}

	if !opts.DryRun {
		_, _, err := publish.PublishLambdaLayers(types.ConvertToPublishOptions(opts), layers)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	app, _ := createApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
