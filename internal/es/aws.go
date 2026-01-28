package es

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/labtiva/stoptail/internal/config"
)

type sigv4Transport struct {
	wrapped http.RoundTripper
	signer  *v4.Signer
	creds   aws.CredentialsProvider
	region  string
	service string
}

func (t *sigv4Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	creds, err := t.creds.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("retrieving AWS credentials: %w", err)
	}

	var body []byte
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	hash := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(hash[:])

	err = t.signer.SignHTTP(req.Context(), creds, req, payloadHash, t.service, t.region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("signing request: %w", err)
	}

	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return t.wrapped.RoundTrip(req)
}

func newAWSTransport(cfg *config.Config) (http.RoundTripper, error) {
	ctx := context.Background()

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.AWSRegion))

	if cfg.AWSProfile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.AWSProfile))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	if _, err := awsCfg.Credentials.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("AWS credentials not found: %w (configure AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY, ~/.aws/credentials, or IAM role)", err)
	}

	return &sigv4Transport{
		wrapped: http.DefaultTransport,
		signer:  v4.NewSigner(),
		creds:   awsCfg.Credentials,
		region:  cfg.AWSRegion,
		service: cfg.AWSService,
	}, nil
}
