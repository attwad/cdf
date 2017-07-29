package upload

import (
	"context"
	"fmt"
	"io"
	"log"

	"cloud.google.com/go/storage"
)

// FileUploader uploads files to a storage service.
type FileUploader interface {
	// UploadFile uploads the file read by the given Reader and give it a name as specified.
	UploadFile(r io.Reader, name string) error
	// Path returns the full path of the file in the storage backend used.
	Path(base string) string
	// Delete deletes the given file.
	Delete(name string) error
}

type gcsFileUploader struct {
	client *storage.Client
	ctx    context.Context
	bucket string
}

// NewGCSFileUploader creates a new FileUploader that uses Google Cloud Storage.
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
	log.Println("Uploading", name, "to bucket", u.bucket)
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
	log.Println("Deleting", name, "from bucket", u.bucket)
	o := u.client.Bucket(u.bucket).Object(name)
	return o.Delete(u.ctx)
}
