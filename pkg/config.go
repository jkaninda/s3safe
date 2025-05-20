/*
 * MIT License
 *
 * Copyright (c) 2025 Jonas Kaninda
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package pkg

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jkaninda/s3safe/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	// Load environment variables from .env file
	_ = godotenv.Load()
}

type Config struct {
	Path          string
	File          string
	Dest          string
	Region        string
	Bucket        string
	KeyID         string
	Secret        string
	EndPoint      string
	ForcePath     bool
	DisableSSL    bool
	Compress      bool
	Decompress    bool
	Timestamp     bool
	IgnoreErrors  bool
	Recursive     bool
	Force         bool
	RetentionDays int
	Exclude       []string
	EnvFile       string
}

type S3Storage struct {
	bucket  string
	session *session.Session
}

type Item struct {
	Key          string
	LastModified time.Time
	IsDir        bool
}

// NewConfig creates a new Config instance from cobra command flags
func NewConfig(cmd *cobra.Command) *Config {
	c := &Config{}

	// Load environment variables first if specified
	c.loadEnvironment(cmd)

	// Load basic flags
	c.loadBasicFlags(cmd)

	// Load AWS configuration
	c.loadAWSConfig()

	// Process path and file configurations
	c.processPaths()

	return c
}

func (c *Config) loadEnvironment(cmd *cobra.Command) {
	c.EnvFile, _ = cmd.Flags().GetString("env-file")
	if c.EnvFile != "" {
		loadEnv(c.EnvFile)
	}
}

func (c *Config) loadBasicFlags(cmd *cobra.Command) {
	c.Path, _ = cmd.Flags().GetString("path")
	c.Dest, _ = cmd.Flags().GetString("dest")
	c.File, _ = cmd.Flags().GetString("file")
	c.Compress, _ = cmd.Flags().GetBool("compress")
	c.Decompress, _ = cmd.Flags().GetBool("decompress")
	c.Timestamp, _ = cmd.Flags().GetBool("timestamp")
	c.IgnoreErrors, _ = cmd.Flags().GetBool("ignore-errors")
	c.Recursive, _ = cmd.Flags().GetBool("recursive")
	c.Force, _ = cmd.Flags().GetBool("force")

	exclude, _ := cmd.Flags().GetString("exclude")
	c.Exclude = strings.Split(exclude, ",")
}

func (c *Config) loadAWSConfig() {
	c.Region = utils.Env(utils.RegionEnv)
	c.KeyID = utils.Env(utils.KeyIDEnv)
	c.Secret = utils.Env(utils.SecretEnv)
	c.EndPoint = utils.Env(utils.EndPointEnv)
	c.ForcePath = utils.Env(utils.ForcePathEnv) == "true"
	c.DisableSSL = utils.Env(utils.DisableSSLEnv) == "true"

	if c.EndPoint == "" {
		c.EndPoint = utils.AwsS3Url
	}

	if c.Bucket == "" {
		c.Bucket = utils.Env(utils.BucketEnv)
	}
}

func (c *Config) processPaths() {
	// Remove trailing slashes
	c.Path = strings.TrimSuffix(c.Path, "/")
	c.Dest = strings.TrimSuffix(c.Dest, "/")

	// Handle file path processing
	if c.File != "" && c.File != "." {
		path := filepath.Join(c.Path, filepath.Dir(c.File))
		file := filepath.Base(c.File)
		c.File = file
		c.Path = path
	}
}

// Validate checks the configuration and ensures all required fields are present
func (c *Config) Validate() error {
	if err := c.validateRequiredFields(); err != nil {
		return err
	}

	return c.validateS3Connection()
}

func (c *Config) validateRequiredFields() error {
	requiredFields := map[string]string{
		c.Region:   "region is required, set AWS_REGION env variable",
		c.Bucket:   "bucket is required, set AWS_BUCKET env variable",
		c.KeyID:    "key id is required, set AWS_ACCESS_KEY_ID env variable",
		c.Secret:   "secret is required, set AWS_SECRET_KEY env variable",
		c.EndPoint: "endpoint is required, set AWS_ENDPOINT env variable",
	}

	for field, errMsg := range requiredFields {
		if field == "" {
			return errors.New(errMsg)
		}
	}

	return nil
}

func (c *Config) validateS3Connection() error {
	s3Storage, err := c.NewS3Storage()
	if err != nil {
		return fmt.Errorf("failed to create S3 storage: %w", err)
	}

	exists, err := bucketExists(s3.New(s3Storage.session), c.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %s does not exist", c.Bucket)
	}

	return nil
}

// NewS3Storage creates a new S3Storage instance from the configuration
func (c *Config) NewS3Storage() (*S3Storage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(c.Region),
		Credentials:      credentials.NewStaticCredentials(c.KeyID, c.Secret, ""),
		Endpoint:         aws.String(c.EndPoint),
		DisableSSL:       aws.Bool(c.DisableSSL),
		S3ForcePathStyle: aws.Bool(c.ForcePath),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create S3 session: %w", err)
	}

	return &S3Storage{
		bucket:  c.Bucket,
		session: sess,
	}, nil
}

func loadEnv(file string) {
	slog.Info("Loading environment variables", "file", file)
	if err := godotenv.Load(file); err != nil {
		slog.Error("Error loading environment variable", "file", file, "error", err)
	}
}

func bucketExists(s3Client *s3.S3, bucket string) (bool, error) {
	_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err == nil {
		return true, nil
	}

	var aErr awserr.RequestFailure
	if errors.As(err, &aErr) && aErr.StatusCode() == http.StatusNotFound {
		return false, nil
	}

	return false, err
}
