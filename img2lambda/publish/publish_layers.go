// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package publish

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
)

func PublishLambdaLayers(opts *types.PublishOptions, layers []types.LambdaLayer) (string, string, error) {
	layerArns := []string{}

	for _, layer := range layers {
		layerName := opts.LayerPrefix + "-" + strings.Replace(layer.Digest, ":", "-", -1)

		var layerDescription, licenseInfo *string

		if opts.Description == "" {
			// if no description is passed from commandline, use the default description
			layerDescription = aws.String("created by img2lambda from image " + opts.SourceImageName)
		} else {
			layerDescription = aws.String(opts.Description)
		}

		if opts.LicenseInfo != "" {
			licenseInfo = aws.String(opts.LicenseInfo)
		}

		if len(opts.CompatibleRuntimes) == 0 {
			opts.CompatibleRuntimes = append(opts.CompatibleRuntimes, "provided")
		}

		layerContents, err := ioutil.ReadFile(layer.File)
		if err != nil {
			return "", "", err
		}

		found, existingArn, err := matchExistingLambdaLayer(layerName, layerContents, &opts.LambdaClient)
		if err != nil {
			return "", "", err
		}

		if found {
			layerArns = append(layerArns, existingArn)
			log.Printf("Matched Lambda layer file %s (image layer %s) to existing Lambda layer: %s", layer.File, layer.Digest, existingArn)
		} else {
			publishArgs := &lambda.PublishLayerVersionInput{
				CompatibleRuntimes: aws.StringSlice(opts.CompatibleRuntimes),
				Content:            &lambda.LayerVersionContentInput{ZipFile: layerContents},
				Description:        layerDescription,
				LayerName:          aws.String(layerName),
				LicenseInfo:        licenseInfo,
			}

			resp, err := opts.LambdaClient.PublishLayerVersion(publishArgs)
			if err != nil {
				return "", "", err
			}

			layerArns = append(layerArns, *resp.LayerVersionArn)
			log.Printf("Published Lambda layer file %s (image layer %s) to Lambda: %s", layer.File, layer.Digest, *resp.LayerVersionArn)
		}

		err = os.Remove(layer.File)
		if err != nil {
			return "", "", err
		}
	}

	jsonArns, err := json.MarshalIndent(layerArns, "", "  ")
	if err != nil {
		return "", "", err
	}

	jsonResultsPath := filepath.Join(opts.ResultsDir, "layers.json")
	jsonFile, err := os.Create(jsonResultsPath)
	if err != nil {
		return "", "", err
	}
	defer jsonFile.Close()

	_, err = jsonFile.Write(jsonArns)
	if err != nil {
		return "", "", err
	}

	yamlArns, err := yaml.Marshal(layerArns)
	if err != nil {
		return "", "", err
	}

	yamlResultsPath := filepath.Join(opts.ResultsDir, "layers.yaml")
	yamlFile, err := os.Create(yamlResultsPath)
	if err != nil {
		return "", "", err
	}
	defer yamlFile.Close()

	_, err = yamlFile.Write(yamlArns)
	if err != nil {
		return "", "", err
	}

	log.Printf("Lambda layer ARNs (%d total) are written to %s and %s", len(layerArns), jsonResultsPath, yamlResultsPath)

	return jsonResultsPath, yamlResultsPath, nil
}

func matchExistingLambdaLayer(layerName string, layerContents []byte, lambdaClient *lambdaiface.LambdaAPI) (bool, string, error) {
	hash := sha256.Sum256(layerContents)
	hashStr := base64.StdEncoding.EncodeToString(hash[:])

	var marker *string
	client := *lambdaClient

	for {
		listArgs := &lambda.ListLayerVersionsInput{
			LayerName: aws.String(layerName),
			Marker:    marker,
		}

		resp, err := client.ListLayerVersions(listArgs)
		if err != nil {
			return false, "", err
		}

		for _, layerVersion := range resp.LayerVersions {
			getArgs := &lambda.GetLayerVersionInput{
				LayerName:     aws.String(layerName),
				VersionNumber: layerVersion.Version,
			}

			layerResp, err := client.GetLayerVersion(getArgs)
			if err != nil {
				return false, "", err
			}

			if *layerResp.Content.CodeSha256 == hashStr && *layerResp.Content.CodeSize == int64(len(layerContents)) {
				return true, *layerResp.LayerVersionArn, nil
			}
		}

		if resp.NextMarker == nil {
			break
		}

		marker = resp.NextMarker
	}

	return false, "", nil
}
