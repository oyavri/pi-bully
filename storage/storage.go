package storage

import "context"

type Storage interface {
	Download(ctx context.Context, uri string, dest string) error
	Upload(ctx context.Context, uri string, src string) error
}
