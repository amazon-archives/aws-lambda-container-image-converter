package publish

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/awslabs/aws-lambda-container-image-converter/img2lambda/types"
)

func PublishLambdaLayers(sourceImageName string, layers []types.LambdaLayer, region string, layerPrefix string, resultsDir string) error {
	layerArns := []string{}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := lambda.New(sess, &aws.Config{Region: aws.String(region)})

	for _, layer := range layers {
		layerName := layerPrefix + "-" + strings.Replace(layer.Digest, ":", "-", -1)

		layerContents, err := ioutil.ReadFile(layer.File)
		if err != nil {
			return err
		}

		publishArgs := &lambda.PublishLayerVersionInput{
			CompatibleRuntimes: []*string{aws.String("provided")},
			Content:            &lambda.LayerVersionContentInput{ZipFile: layerContents},
			Description:        aws.String("created by img2lambda from image " + sourceImageName),
			LayerName:          aws.String(layerName),
		}

		resp, err := svc.PublishLayerVersion(publishArgs)
		if err != nil {
			return err
		}

		layerArns = append(layerArns, *resp.LayerVersionArn)

		err = os.Remove(layer.File)
		if err != nil {
			return err
		}

		log.Printf("Published Lambda layer file %s (image layer %s) to Lambda: %s", layer.File, layer.Digest, *resp.LayerVersionArn)
	}

	jsonArns, err := json.MarshalIndent(layerArns, "", "  ")
	if err != nil {
		return err
	}

	resultsPath := filepath.Join(resultsDir, "layers.json")
	jsonFile, err := os.Create(resultsPath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	_, err = jsonFile.Write(jsonArns)
	if err != nil {
		return err
	}

	log.Printf("Lambda layer ARNs (%d total) are written to %s", len(layerArns), resultsPath)

	return nil
}
