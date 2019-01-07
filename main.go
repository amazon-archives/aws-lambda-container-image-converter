package main

import (
	"log"
	"os"

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
	app.Usage = "Repackages a container image into Lambda layers and publishes them to Lambda"
	app.Action = func(c *cli.Context) error {
		return repackImageAction(&opts)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "image, i",
			Usage:       "Name of the source container image. Format is 'transport:details', such as 'docker-daemon:my-docker-image:latest",
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
			Usage:       "Conduct a dry-run: Repackage the image, but write the Lambda layers to local disk (do not publish to Lambda)",
			Destination: &opts.dryRun,
		},
	}
	return app, &opts
}

func repackImageAction(opts *cmdOptions) error {
	return RepackImage(opts.image)
}

func main() {
	app, _ := createApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
