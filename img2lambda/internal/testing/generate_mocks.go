package testing

//go:generate mockgen.sh github.com/aws/aws-sdk-go/service/lambda/lambdaiface LambdaAPI mocks/lambda_mocks.go
//go:generate mockgen.sh github.com/containers/image/types ImageCloser,ImageSource mocks/image_mocks.go
