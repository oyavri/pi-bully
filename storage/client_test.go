package storage_test

import (
	"context"
	"os"
	"testing"

	"github.com/oyavri/pi-bully/config"
	"github.com/oyavri/pi-bully/storage"
	"go.uber.org/zap"
)

func TestUploadAndDownload(t *testing.T) {
	cfg := config.StorageConfig{
		Endpoint:  "localhost:9000",
		Bucket:    "bully",
		AccessKey: "test",
		SecretKey: "test",
		Region:    "us-east-1",
		UseSSL:    false,
	}

	logger, _ := zap.NewDevelopment()
	client, err := storage.New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tmp, err := os.CreateTemp("", "test-upload-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("hello from pi_bully")
	tmp.Close()

	if err := client.Upload(context.Background(), "s3://bully/test/hello.txt", tmp.Name()); err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	dest := tmp.Name() + ".downloaded"
	defer os.Remove(dest)
	if err := client.Download(context.Background(), "s3://bully/test/hello.txt", dest); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) != "hello from pi_bully" {
		t.Fatalf("expected 'hello from pi_bully', got %q", string(content))
	}
}
