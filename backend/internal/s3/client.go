package s3

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// settings holds the resolved object storage configuration.
type settings struct {
	bucket   string
	region   string
	endpoint string
}

func loadSettings() settings {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	bucket := os.Getenv("AWS_S3_BUCKET")
	if bucket == "" {
		bucket = "orcs-deployments"
	}
	return settings{
		bucket:   bucket,
		region:   region,
		endpoint: os.Getenv("AWS_S3_ENDPOINT"),
	}
}

// newClient builds an S3 client from the environment along with the resolved settings.
func newClient(ctx context.Context) (*s3.Client, settings, error) {
	set := loadSettings()
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	var cfg aws.Config
	var err error

	if accessKey != "" && secretKey != "" {
		opts := []func(*config.LoadOptions) error{
			config.WithRegion(set.region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		}
		if set.endpoint != "" {
			opts = append(opts, config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, reg string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: set.endpoint, SigningRegion: reg}, nil
				},
			)))
		}
		cfg, err = config.LoadDefaultConfig(ctx, opts...)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(set.region))
	}

	if err != nil {
		return nil, set, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if set.endpoint != "" {
			o.UsePathStyle = true
		}
	})

	return client, set, nil
}

// ensureBucket creates the bucket if it is missing and makes its objects world readable,
// so a deployed site can be opened straight from the returned URL.
func ensureBucket(ctx context.Context, client *s3.Client, set settings) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(set.bucket)})
	if err != nil {
		fmt.Printf("[S3] Bucket %q does not exist, creating it...\n", set.bucket)
		if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(set.bucket)}); err != nil {
			return fmt.Errorf("failed to create bucket %q: %w", set.bucket, err)
		}
		fmt.Printf("[S3] Bucket %q successfully created!\n", set.bucket)
	}

	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, set.bucket)
	if _, err := client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(set.bucket),
		Policy: aws.String(policy),
	}); err != nil {
		// Not fatal: the files are uploaded either way, they just may not be publicly readable
		fmt.Printf("[S3] Warning: could not apply public-read policy to %q: %s\n", set.bucket, err.Error())
	}

	return nil
}

// contentTypeFor resolves a content type for a key, falling back to a small table for the
// web extensions that matter most when serving a built site.
func contentTypeFor(key string) string {
	ext := strings.ToLower(path.Ext(key))
	switch ext {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".json", ".map":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".woff2":
		return "font/woff2"
	}
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// stripPrefix removes the first stripComponents path segments from a tar entry name,
// so "dist/assets/app.js" is published as "assets/app.js".
func stripPrefix(name string, stripComponents int) string {
	name = strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(name, "\\", "/")), "/")
	parts := strings.Split(name, "/")
	if len(parts) <= stripComponents {
		return ""
	}
	return strings.Join(parts[stripComponents:], "/")
}

// UploadTarStream reads a tar archive and uploads every regular file it contains under
// s3Prefix. It returns how many files were written, so an empty archive (a build output
// directory that does not exist) can be told apart from a successful upload.
func UploadTarStream(ctx context.Context, r io.Reader, s3Prefix string, stripComponents int) (int, error) {
	client, set, err := newClient(ctx)
	if err != nil {
		return 0, err
	}
	if err := ensureBucket(ctx, client, set); err != nil {
		return 0, err
	}

	prefix := strings.Trim(s3Prefix, "/")
	uploaded := 0
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return uploaded, fmt.Errorf("failed to read build archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		relPath := stripPrefix(header.Name, stripComponents)
		if relPath == "" {
			continue
		}

		// PutObject needs a seekable body to sign the payload, and build assets are small
		// enough to stage in memory one at a time.
		body, err := io.ReadAll(tr)
		if err != nil {
			return uploaded, fmt.Errorf("failed to read %s from build archive: %w", header.Name, err)
		}

		key := prefix + "/" + relPath
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(set.bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(body),
			ContentType: aws.String(contentTypeFor(key)),
		})
		if err != nil {
			return uploaded, fmt.Errorf("failed to upload %s: %w", key, err)
		}
		uploaded++
	}

	return uploaded, nil
}

// PublicURL builds a browser reachable URL for an uploaded object.
func PublicURL(key string) string {
	set := loadSettings()
	key = strings.TrimPrefix(key, "/")

	if set.endpoint != "" {
		// The browser cannot resolve docker network hostnames, so publish the host mapping
		publicEndpoint := strings.TrimSuffix(set.endpoint, "/")
		publicEndpoint = strings.Replace(publicEndpoint, "minio:9000", "localhost:9000", 1)
		return fmt.Sprintf("%s/%s/%s", publicEndpoint, set.bucket, key)
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", set.bucket, set.region, key)
}

// Describe reports the object storage target this process will publish to. Used at startup
// so a misconfigured endpoint is obvious before the first build finishes.
func Describe() string {
	set := loadSettings()
	if set.endpoint == "" {
		return fmt.Sprintf("bucket %q on AWS S3 (%s)", set.bucket, set.region)
	}
	return fmt.Sprintf("bucket %q at %s", set.bucket, set.endpoint)
}

// DeletePrefix removes every object stored under the given prefix. Used when a deployment is deleted.
func DeletePrefix(ctx context.Context, s3Prefix string) error {
	client, set, err := newClient(ctx)
	if err != nil {
		return err
	}

	prefix := strings.TrimSuffix(s3Prefix, "/") + "/"
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(set.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects under %q: %w", prefix, err)
		}

		objects := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
		}
		if len(objects) == 0 {
			continue
		}

		_, err = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(set.bucket),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects under %q: %w", prefix, err)
		}
	}

	return nil
}
