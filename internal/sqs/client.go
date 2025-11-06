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
	configOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	// Override endpoint for LocalStack if specified
	if endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			// Only override endpoint for SQS service
			if service == sqs.ServiceID {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: region,
				}, nil
			}
			// Return default endpoint for other services
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		configOptions = append(configOptions, awsconfig.WithEndpointResolverWithOptions(customResolver))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return nil, err
	}

	return sqs.NewFromConfig(awsCfg), nil
}
