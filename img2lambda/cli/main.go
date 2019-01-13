package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/extract"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/publish"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/version"
	"github.com/urfave/cli"
)

type cmdOptions struct {
	image          string // Name of the container image
	region         string // AWS region
	outputDir      string // Output directory for the Lambda layers
	dryRun         bool   // Dry-run (will not register with Lambda)
	layerNamespace string // Prefix for published Lambda layers
}

func createApp() (*cli.App, *cmdOptions) {
	opts := cmdOptions{}

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "img2lambda"
	app.Version = version.VersionString()
	app.Usage = "Repackages a container image into AWS Lambda layers and publishes them to Lambda"
	app.Action = func(c *cli.Context) error {
		validateCliOptions(&opts, c)
		return repackImageAction(&opts)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "image, i",
			Usage:       "Name of the source container image. For example, 'my-docker-image:latest'. The Docker daemon must be pulled locally already.",
			Destination: &opts.image,
		},
		cli.StringFlag{
			Name:        "region, r",
			Usage:       "AWS region",
			Value:       "us-east-1",
			Destination: &opts.region,
		},
		cli.StringFlag{
			Name:        "output-directory, o",
			Usage:       "Destination directory for command output",
			Value:       "./output",
			Destination: &opts.outputDir,
		},
		cli.StringFlag{
			Name:        "layer-namespace, n",
			Usage:       "Prefix for the layers published to Lambda",
			Value:       "img2lambda",
			Destination: &opts.layerNamespace,
		},
		cli.BoolFlag{
			Name:        "dry-run, d",
			Usage:       "Conduct a dry-run: Repackage the image, but only write the Lambda layers to local disk (do not publish to Lambda)",
			Destination: &opts.dryRun,
		},
	}

	app.Setup()
	app.Commands = []cli.Command{}

	return app, &opts
}

func validateCliOptions(opts *cmdOptions, context *cli.Context) {
	if opts.image == "" {
		fmt.Println("ERROR: Image name is required\n")
		cli.ShowAppHelpAndExit(context, 1)
	}
}

func repackImageAction(opts *cmdOptions) error {
	layers, err := extract.RepackImage("docker-daemon:"+opts.image, opts.outputDir)
	if err != nil {
		return err
	}

	if len(layers) == 0 {
		return errors.New("No compatible layers found in the image (likely nothing found in /opt)")
	}

	if !opts.dryRun {
		err := publish.PublishLambdaLayers(opts.image, layers, opts.region, opts.layerNamespace, opts.outputDir)
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
