package s3_test

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/santekno/sdk/storagex"
	"github.com/santekno/sdk/storagex/s3"
)

// A client per bucket and credential set; variants are written once and never
// replaced, and browsers upload originals with a presigned PUT.
func Example() {
	media, err := s3.New(s3.Config{
		Endpoint:        os.Getenv("MEDIA_S3_ENDPOINT"), // https://<project>.supabase.co/storage/v1/s3
		Region:          os.Getenv("MEDIA_S3_REGION"),
		Bucket:          "media",
		AccessKeyID:     os.Getenv("MEDIA_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("MEDIA_S3_SECRET_ACCESS_KEY"),
	}, storagex.WithNoOverwrite())
	if err != nil {
		log.Fatal(err) // names the bad field, never its value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = media.Put(ctx, "v/0123456789abcdef-480.webp", "image/webp", strings.NewReader("..."))
	if err != nil && !errors.Is(err, storagex.ErrAlreadyExists) {
		log.Print(err) // operation, status, and S3 code only
	}

	upload, err := media.PresignPut(ctx, "u/0b5c1f0e", "image/jpeg", 10*time.Minute)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(upload) // query redacted; send upload.URL and upload.Header to the browser only
}
