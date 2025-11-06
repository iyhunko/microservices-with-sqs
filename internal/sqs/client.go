package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// NewClient creates and configures a new AWS SQS client.
// It loads the AWS configuration from the environment and optionally sets a custom endpoint.
func NewClient(ctx context.Context, region string, endpoint string) (*sqs.Client, error) {
	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	// Override endpoint for LocalStack if specified
	if endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(endpoint)
	}

	return sqs.NewFromConfig(awsCfg), nil
}
