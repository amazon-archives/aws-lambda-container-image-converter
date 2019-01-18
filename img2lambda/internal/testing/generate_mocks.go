// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package testing

//go:generate mockgen.sh github.com/aws/aws-sdk-go/service/lambda/lambdaiface LambdaAPI mocks/lambda_mocks.go
//go:generate mockgen.sh github.com/containers/image/types ImageCloser,ImageSource mocks/image_mocks.go
