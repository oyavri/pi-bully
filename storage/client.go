package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/oyavri/pi-bully/config"
	"go.uber.org/zap"
)

type Client struct {
	s3     *s3.Client
	bucket string
	logger *zap.Logger
}

func New(cfg config.StorageConfig, logger *zap.Logger) (*Client, error) {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)),
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		UsePathStyle: true, // required for LocalStack
	})

	return &Client{
		s3:     s3Client,
		bucket: cfg.Bucket,
		logger: logger,
	}, nil
}

// Download an object from S3 to a local file path
func (c *Client) Download(ctx context.Context, uri string, dest string) error {
	key := parseKey(uri)

	c.logger.Info("downloading from storage",
		zap.String("uri", uri),
		zap.String("dest", dest),
	)

	resp, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: get object %q: %w", uri, err)
	}
	defer resp.Body.Close()

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("storage: create file %q, %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("storage: write file %q: %w", dest, err)
	}

	c.logger.Info("download complete", zap.String("dest", dest))
	return nil
}

// Upload a local file to S3
func (c *Client) Upload(ctx context.Context, uri string, src string) error {
	key := parseKey(uri)

	c.logger.Info("uploading to storage",
		zap.String("uri", uri),
		zap.String("src", src),
	)

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("storage: open file %q: %w", src, err)
	}
	defer f.Close()

	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("storage: put object %q: %w", uri, err)
	}

	c.logger.Info("upload complete", zap.String("uri", uri))
	return nil
}

// Strip s3://bucket/ prefix from URI and return the key
// s3://bucket/key/path -> key/path
func parseKey(uri string) string {
	if strings.HasPrefix(uri, "s3://") {
		parts := strings.SplitN(uri[5:], "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return uri
}
