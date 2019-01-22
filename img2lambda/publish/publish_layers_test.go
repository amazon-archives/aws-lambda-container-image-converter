// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package publish

import (
	"encoding/json"
	"errors"
	"fmt"
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
	layers = append(layers, mockLayer(t, 3))

	return layers
}

func mockPublishNoExistingLayers(t *testing.T, lambdaClient *mocks.MockLambdaAPI, n int) {
	layerName := aws.String(fmt.Sprintf("test-prefix-sha256-%d", n))

	expectedPublishInput := &lambda.PublishLayerVersionInput{
		CompatibleRuntimes: []*string{aws.String("provided")},
		Content:            &lambda.LayerVersionContentInput{ZipFile: []byte(fmt.Sprintf("hello world %d", n))},
		Description:        aws.String("created by img2lambda from image test-image"),
		LayerName:          layerName,
	}

	expectedPublishOutput := &lambda.PublishLayerVersionOutput{
		LayerVersionArn: aws.String(fmt.Sprintf("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-%d:1", n)),
	}

	// Test out pagination
	expectedListInput1 := &lambda.ListLayerVersionsInput{
		LayerName: layerName,
	}

	expectedListOutput1 := &lambda.ListLayerVersionsOutput{
		LayerVersions: []*lambda.LayerVersionsListItem{},
		NextMarker:    aws.String("hello"),
	}

	expectedListInput2 := &lambda.ListLayerVersionsInput{
		LayerName: layerName,
		Marker:    aws.String("hello"),
	}

	expectedListOutput2 := &lambda.ListLayerVersionsOutput{
		LayerVersions: []*lambda.LayerVersionsListItem{},
	}

	gomock.InOrder(
		lambdaClient.EXPECT().ListLayerVersions(gomock.Eq(expectedListInput1)).Return(expectedListOutput1, nil),
		lambdaClient.EXPECT().ListLayerVersions(gomock.Eq(expectedListInput2)).Return(expectedListOutput2, nil),
		lambdaClient.EXPECT().PublishLayerVersion(gomock.Eq(expectedPublishInput)).Return(expectedPublishOutput, nil),
	)
}

func mockPublishNoMatchingLayers(t *testing.T, lambdaClient *mocks.MockLambdaAPI, n int) {
	layerName := aws.String(fmt.Sprintf("test-prefix-sha256-%d", n))

	expectedPublishInput := &lambda.PublishLayerVersionInput{
		CompatibleRuntimes: []*string{aws.String("provided")},
		Content:            &lambda.LayerVersionContentInput{ZipFile: []byte(fmt.Sprintf("hello world %d", n))},
		Description:        aws.String("created by img2lambda from image test-image"),
		LayerName:          layerName,
	}

	expectedPublishOutput := &lambda.PublishLayerVersionOutput{
		LayerVersionArn: aws.String(fmt.Sprintf("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-%d:1", n)),
	}

	expectedListInput := &lambda.ListLayerVersionsInput{
		LayerName: layerName,
	}

	var existingVersions []*lambda.LayerVersionsListItem
	existingVersionNumber := int64(0)
	existingVersionListItem := &lambda.LayerVersionsListItem{
		LayerVersionArn: aws.String(fmt.Sprintf("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-%d:0", n)),
		Version:         &existingVersionNumber,
	}
	existingVersions = append(existingVersions, existingVersionListItem)
	expectedListOutput := &lambda.ListLayerVersionsOutput{
		LayerVersions: existingVersions,
	}

	expectedGetInput := &lambda.GetLayerVersionInput{
		LayerName:     layerName,
		VersionNumber: &existingVersionNumber,
	}

	size := int64(0)
	expectedContentOutput := &lambda.LayerVersionContentOutput{
		CodeSha256: aws.String("kjsdflkjfd"),
		CodeSize:   &size,
	}

	expectedGetOutput := &lambda.GetLayerVersionOutput{
		Version:         &existingVersionNumber,
		Content:         expectedContentOutput,
		LayerVersionArn: aws.String(fmt.Sprintf("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-%d:0", n)),
	}

	gomock.InOrder(
		lambdaClient.EXPECT().ListLayerVersions(gomock.Eq(expectedListInput)).Return(expectedListOutput, nil),
		lambdaClient.EXPECT().GetLayerVersion(gomock.Eq(expectedGetInput)).Return(expectedGetOutput, nil),
		lambdaClient.EXPECT().PublishLayerVersion(gomock.Eq(expectedPublishInput)).Return(expectedPublishOutput, nil),
	)
}

func mockMatchingLayer(t *testing.T, lambdaClient *mocks.MockLambdaAPI, n int) {
	layerName := aws.String(fmt.Sprintf("test-prefix-sha256-%d", n))

	expectedListInput := &lambda.ListLayerVersionsInput{
		LayerName: layerName,
	}

	var existingVersions []*lambda.LayerVersionsListItem
	existingVersionNumber := int64(0)
	existingVersionListItem := &lambda.LayerVersionsListItem{
		LayerVersionArn: aws.String(fmt.Sprintf("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-%d:1", n)),
		Version:         &existingVersionNumber,
	}
	existingVersions = append(existingVersions, existingVersionListItem)
	expectedListOutput := &lambda.ListLayerVersionsOutput{
		LayerVersions: existingVersions,
	}

	expectedGetInput := &lambda.GetLayerVersionInput{
		LayerName:     layerName,
		VersionNumber: &existingVersionNumber,
	}

	size := int64(13)
	expectedContentOutput := &lambda.LayerVersionContentOutput{
		CodeSha256: aws.String("T/q7q052MgJGLfH1mBGUQSFYjwVn9VvOWBoOmevPZgY="),
		CodeSize:   &size,
	}

	expectedGetOutput := &lambda.GetLayerVersionOutput{
		Version:         &existingVersionNumber,
		Content:         expectedContentOutput,
		LayerVersionArn: aws.String(fmt.Sprintf("arn:aws:lambda:us-east-2:123456789012:layer:example-layer-%d:1", n)),
	}

	gomock.InOrder(
		lambdaClient.EXPECT().ListLayerVersions(gomock.Eq(expectedListInput)).Return(expectedListOutput, nil),
		lambdaClient.EXPECT().GetLayerVersion(gomock.Eq(expectedGetInput)).Return(expectedGetOutput, nil),
	)
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

	mockPublishNoExistingLayers(t, lambdaClient, 1)
	mockPublishNoMatchingLayers(t, lambdaClient, 2)
	mockMatchingLayer(t, lambdaClient, 3)

	resultsFilename, err := PublishLambdaLayers(opts, layers)
	assert.Nil(t, err)

	resultArns := parseResult(t, resultsFilename)
	assert.Len(t, resultArns, 3)
	assert.Equal(t, "arn:aws:lambda:us-east-2:123456789012:layer:example-layer-1:1", resultArns[0])
	assert.Equal(t, "arn:aws:lambda:us-east-2:123456789012:layer:example-layer-2:1", resultArns[1])
	assert.Equal(t, "arn:aws:lambda:us-east-2:123456789012:layer:example-layer-3:1", resultArns[2])

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

	expectedListInput := &lambda.ListLayerVersionsInput{
		LayerName: aws.String("test-prefix-sha256-1"),
	}

	expectedListOutput := &lambda.ListLayerVersionsOutput{
		LayerVersions: []*lambda.LayerVersionsListItem{},
	}

	lambdaClient.EXPECT().ListLayerVersions(gomock.Eq(expectedListInput)).Return(expectedListOutput, nil)

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
	os.Remove(layers[2].File)

	os.Remove(dir)
}
