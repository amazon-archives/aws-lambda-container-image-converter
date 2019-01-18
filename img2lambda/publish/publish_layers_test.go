// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package publish

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/internal/testing/mocks"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func parseResult(t *testing.T, resultsFilename string) []string {
	resultContents, err := ioutil.ReadFile(resultsFilename)
	assert.Nil(t, err)
	var resultArns []string
	err = json.Unmarshal(resultContents, &resultArns)
	assert.Nil(t, err)
	os.Remove(resultsFilename)
	return resultArns
}

func mockLayer(t *testing.T, n int) types.LambdaLayer {
	tmpFile, err := ioutil.TempFile("", "")
	assert.Nil(t, err)
	defer tmpFile.Close()

	_, err = tmpFile.WriteString("hello world " + strconv.Itoa(n))
	assert.Nil(t, err)
	return types.LambdaLayer{
		Digest: "sha256:" + strconv.Itoa(n),
		File:   tmpFile.Name(),
	}
}

func mockLayers(t *testing.T) []types.LambdaLayer {
	var layers []types.LambdaLayer

	layers = append(layers, mockLayer(t, 1))
	layers = append(layers, mockLayer(t, 2))

	return layers
}

func TestNoLayers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lambdaClient := mocks.NewMockLambdaAPI(ctrl)

	dir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)

	opts := &types.PublishOptions{
		LambdaClient:    lambdaClient,
		LayerPrefix:     "test-prefix",
		SourceImageName: "test-image",
		ResultsDir:      dir,
	}

	layers := []types.LambdaLayer{}

	resultsFilename, err := PublishLambdaLayers(opts, layers)
	assert.Nil(t, err)

	resultArns := parseResult(t, resultsFilename)
	assert.Len(t, resultArns, 0)

	os.Remove(dir)
}

func TestPublishSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lambdaClient := mocks.NewMockLambdaAPI(ctrl)

	dir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)

	opts := &types.PublishOptions{
		LambdaClient:    lambdaClient,
		LayerPrefix:     "test-prefix",
		SourceImageName: "test-image",
		ResultsDir:      dir,
	}

	layers := mockLayers(t)

	expectedInput1 := &lambda.PublishLayerVersionInput{
		CompatibleRuntimes: []*string{aws.String("provided")},
		Content:            &lambda.LayerVersionContentInput{ZipFile: []byte("hello world 1")},
		Description:        aws.String("created by img2lambda from image test-image"),
		LayerName:          aws.String("test-prefix-sha256-1"),
	}

	expectedOutput1 := &lambda.PublishLayerVersionOutput{
		LayerVersionArn: aws.String("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-1:1"),
	}

	expectedInput2 := &lambda.PublishLayerVersionInput{
		CompatibleRuntimes: []*string{aws.String("provided")},
		Content:            &lambda.LayerVersionContentInput{ZipFile: []byte("hello world 2")},
		Description:        aws.String("created by img2lambda from image test-image"),
		LayerName:          aws.String("test-prefix-sha256-2"),
	}

	expectedOutput2 := &lambda.PublishLayerVersionOutput{
		LayerVersionArn: aws.String("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-2:1"),
	}

	lambdaClient.EXPECT().
		PublishLayerVersion(gomock.Eq(expectedInput1)).
		Return(expectedOutput1, nil)

	lambdaClient.EXPECT().
		PublishLayerVersion(gomock.Eq(expectedInput2)).
		Return(expectedOutput2, nil)

	resultsFilename, err := PublishLambdaLayers(opts, layers)
	assert.Nil(t, err)

	resultArns := parseResult(t, resultsFilename)
	assert.Len(t, resultArns, 2)
	assert.Equal(t, "arn:aws:lambda:us-east-2:123456789012:layer:example-layer-1:1", resultArns[0])
	assert.Equal(t, "arn:aws:lambda:us-east-2:123456789012:layer:example-layer-2:1", resultArns[1])

	os.Remove(dir)
}

func TestPublishError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lambdaClient := mocks.NewMockLambdaAPI(ctrl)

	dir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)

	opts := &types.PublishOptions{
		LambdaClient:    lambdaClient,
		LayerPrefix:     "test-prefix",
		SourceImageName: "test-image",
		ResultsDir:      dir,
	}

	layers := mockLayers(t)

	expectedInput1 := &lambda.PublishLayerVersionInput{
		CompatibleRuntimes: []*string{aws.String("provided")},
		Content:            &lambda.LayerVersionContentInput{ZipFile: []byte("hello world 1")},
		Description:        aws.String("created by img2lambda from image test-image"),
		LayerName:          aws.String("test-prefix-sha256-1"),
	}

	lambdaClient.EXPECT().
		PublishLayerVersion(gomock.Eq(expectedInput1)).
		Return(nil, errors.New("Access denied"))

	resultsFilename, err := PublishLambdaLayers(opts, layers)
	assert.Error(t, err)
	assert.Equal(t, "", resultsFilename)

	os.Remove(layers[0].File)
	os.Remove(layers[1].File)

	os.Remove(dir)
}
