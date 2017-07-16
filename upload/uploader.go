package upload

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

type FileUploader interface {
	UploadFile(r io.Reader, name string) error
	Path(base string) string
	Delete(name string) error
}

type gcsFileUploader struct {
	client *storage.Client
	ctx    context.Context
	bucket string
}

func NewGCSFileUploader(bucket string) (FileUploader, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &gcsFileUploader{
		client: client,
		ctx:    ctx,
		bucket: bucket,
	}, nil
}

func (u *gcsFileUploader) Path(base string) string {
	return fmt.Sprintf("gs://%s/%s", u.bucket, base)
}

func (u *gcsFileUploader) UploadFile(r io.Reader, name string) error {
	bkt := u.client.Bucket(u.bucket)
	w := bkt.Object(name).NewWriter(u.ctx)
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("copy: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close: %v", err)
	}
	return nil
}

func (u *gcsFileUploader) Delete(name string) error {
	o := u.client.Bucket(u.bucket).Object(name)
	return o.Delete(u.ctx)
}
