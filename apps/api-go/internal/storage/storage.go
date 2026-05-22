package storage

import (
	"context"
	"io"
	"net/url"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Store interface {
	Put(ctx context.Context, key string, contentType string, contentLength int64, input io.Reader) error
	Get(ctx context.Context, key string) (StoredAudio, error)
}

type StoredAudio struct {
	Bytes         io.ReadCloser
	ContentType   string
	ContentLength *int64
}

type S3Store struct {
	bucket string
	client *s3.Client
}

func NewS3Store(ctx context.Context, cfg config.S3) (*S3Store, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
			options.UsePathStyle = true
		}
	})
	store := &S3Store{bucket: cfg.Bucket, client: client}
	if cfg.Endpoint != "" {
		if err := store.ensureBucket(ctx); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *S3Store) ensureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)})
	return err
}

func (s *S3Store) Put(ctx context.Context, key string, contentType string, contentLength int64, input io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
		Body:          input,
	})
	return err
}

func (s *S3Store) Get(ctx context.Context, key string) (StoredAudio, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return StoredAudio{}, err
	}
	contentType := "application/octet-stream"
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return StoredAudio{Bytes: out.Body, ContentType: contentType, ContentLength: out.ContentLength}, nil
}

func ExtensionForMimeType(mimeType string) string {
	switch {
	case mimeType == "audio/mp4":
		return "mp4"
	case mimeType == "audio/mpeg":
		return "mp3"
	case mimeType == "audio/wav":
		return "wav"
	case mimeType == "audio/ogg":
		return "ogg"
	default:
		return "webm"
	}
}

func SafeEndpointHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "aws"
	}
	return u.Host
}
