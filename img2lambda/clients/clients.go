// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package clients

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/version"
)

var userAgentHandler = request.NamedHandler{
	Name: "img2lambda.UserAgentHandler",
	Fn:   request.MakeAddToUserAgentHandler("aws-lambda-container-image-converter", version.Version),
}

func NewLambdaClient(region string, profile string) *lambda.Lambda {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Profile: profile,
		SharedConfigState: session.SharedConfigEnable,
	}))
	sess.Handlers.Build.PushBackNamed(userAgentHandler)

	client := lambda.New(sess, &aws.Config{Region: aws.String(region)})

	return client
}
