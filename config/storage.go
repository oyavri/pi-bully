package config

import (
	"fmt"
	"os"
)

type StorageConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

func loadStorageConfig() (StorageConfig, error) {
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET")
	accessKey := os.Getenv("STORAGE_ACCESS_KEY")
	secretKey := os.Getenv("STORAGE_SECRET_KEY")
	region := os.Getenv("STORAGE_REGION")

	if endpoint == "" {
		return StorageConfig{}, fmt.Errorf("STORAGE_ENDPOINT is required")
	}

	if bucket == "" {
		return StorageConfig{}, fmt.Errorf("STORAGE_BUCKET is required")
	}

	if accessKey == "" {
		return StorageConfig{}, fmt.Errorf("STORAGE_ACCESS_KEY is required")
	}

	if secretKey == "" {
		return StorageConfig{}, fmt.Errorf("STORAGE_SECRET_KEY is required")
	}

	if region == "" {
		return StorageConfig{}, fmt.Errorf("STORAGE_REGION is required")
	}

	return StorageConfig{
		Endpoint:  endpoint,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
		UseSSL:    os.Getenv("STORAGE_USE_SSL") == "true",
	}, nil
}
