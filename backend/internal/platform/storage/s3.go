// Package storage stores files in any S3-compatible object storage (RustFS
// locally, any S3 provider in production), chosen by configuration only.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/config"
)

// S3 stores private JPEG files in one bucket and hands out signed links to
// read them.
type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// NewS3 builds a client for the bucket described by cfg. Signed links use
// cfg.PublicEndpoint, because the host is part of the signature.
func NewS3(cfg config.S3) *S3 {
	public := cfg.PublicEndpoint
	if public == "" {
		public = cfg.Endpoint
	}
	return &S3{
		client:  newClient(cfg, cfg.Endpoint),
		presign: s3.NewPresignClient(newClient(cfg, public)),
		bucket:  cfg.Bucket,
	}
}

func newClient(cfg config.S3, endpoint string) *s3.Client {
	return s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		UsePathStyle: cfg.ForcePathStyle,
	})
}

// Put stores data under key as a private image/jpeg object.
func (s *S3) Put(ctx context.Context, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("image/jpeg"),
		ACL:         types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return fmt.Errorf("put %q: %w", key, err)
	}
	return nil
}

// Delete removes the objects stored under keys. Missing keys are not an
// error.
func (s *S3) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	ids := make([]types.ObjectIdentifier, len(keys))
	for i, k := range keys {
		ids[i] = types.ObjectIdentifier{Key: aws.String(k)}
	}
	out, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return fmt.Errorf("delete %d objects: %w", len(keys), err)
	}
	if len(out.Errors) > 0 {
		return fmt.Errorf("delete %q: %s", aws.ToString(out.Errors[0].Key), aws.ToString(out.Errors[0].Message))
	}
	return nil
}

// URL returns a signed link that lets anyone read the object under key until
// ttl passes. It is computed locally: it does not check that the object exists.
func (s *S3) URL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("sign %q: %w", key, err)
	}
	return req.URL, nil
}
