package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/tnqbao/gau-cdn-service/config"
)

type CloudflareR2Client struct {
	Client     *s3.Client
	BucketName string
}

func NewCloudflareR2Client(cfg *appconfig.EnvConfig) (*CloudflareR2Client, error) {
	endpoint := cfg.CloudflareR2.Endpoint
	bucketName := cfg.CloudflareR2.BucketName
	accessKey := cfg.CloudflareR2.AccessKey
	secret := cfg.CloudflareR2.SecretKey

	// Create a custom endpoint resolver for Cloudflare R2 || because int not use default AWS endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			SigningRegion: "auto",
		}, nil
	})

	// Load AWS configuration with custom endpoint and credentials
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secret, "")),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	return &CloudflareR2Client{
		Client:     s3Client,
		BucketName: bucketName,
	}, nil
}

func (r *CloudflareR2Client) GetObject(ctx context.Context, key string) ([]byte, string, error) {
	resp, err := r.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object: %w", err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, "", err
	}

	contentType := aws.ToString(resp.ContentType)
	return buf.Bytes(), contentType, nil
}
