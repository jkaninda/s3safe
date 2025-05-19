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

func (c *Config) NewConfig(cmd *cobra.Command) *Config {
	c.EnvFile, _ = cmd.Flags().GetString("env-file")
	// Check if the EnvFile is set
	if c.EnvFile != "" {
		loadEnv(c.EnvFile)
	}
	c.Path, _ = cmd.Flags().GetString("path")
	c.Dest, _ = cmd.Flags().GetString("dest")
	c.File, _ = cmd.Flags().GetString("file")
	c.Compress, _ = cmd.Flags().GetBool("compress")
	c.Decompress, _ = cmd.Flags().GetBool("decompress")
	c.Timestamp, _ = cmd.Flags().GetBool("timestamp")
	c.IgnoreErrors, _ = cmd.Flags().GetBool("ignore-errors")
	c.Recursive, _ = cmd.Flags().GetBool("recursive")
	exclude, _ := cmd.Flags().GetString("exclude")
	c.Exclude = strings.Split(exclude, ",")
	c.Region = utils.Env(utils.RegionEnv)
	c.Bucket = utils.Env(utils.BucketEnv)
	c.KeyID = utils.Env(utils.KeyIDEnv)
	c.Secret = utils.Env(utils.SecretEnv)
	c.EndPoint = utils.Env(utils.EndPointEnv)
	c.ForcePath = utils.Env(utils.ForcePathEnv) == "true"
	c.DisableSSL = utils.Env(utils.DisableSSLEnv) == "true"
	// Remove trailing slash
	if len(c.Path) > 0 && c.Path[len(c.Path)-1] == '/' {
		c.Path = c.Path[:len(c.Path)-1]
	}
	// Remove trailing slash
	if len(c.Dest) > 0 && c.Dest[len(c.Dest)-1] == '/' {
		c.Dest = c.Dest[:len(c.Dest)-1]
	}
	if c.File != "" && c.File != "." {
		path := filepath.Join(c.Path, filepath.Dir(c.File))
		file := filepath.Base(c.File)
		c.File = file
		c.Path = path
	}
	return c
}
func (c *Config) validate() error {
	if c.Region == "" {
		return fmt.Errorf("region is required, set AWS_REGION env variable")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required, set AWS_BUCKET env variable")
	}
	if c.KeyID == "" {
		return fmt.Errorf("key id is required, set AWS_ACCESS_KEY_ID env variable")
	}
	if c.Secret == "" {
		return fmt.Errorf("secret is required, set AWS_SECRET_KEY env variable")
	}
	if c.EndPoint == "" {
		return fmt.Errorf("endpoint is required, set AWS_ENDPOINT env variable")
	}
	// Note: You had a duplicate check for EndPoint, I removed it

	// Validate S3 connection
	s3Storage, err := c.newS3Storage()
	if err != nil {
		return fmt.Errorf("failed to create S3 storage: %w", err)
	}
	if _, err = bucketExists(s3.New(s3Storage.session), c.Bucket); err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return nil
}
func (c *Config) newS3Storage() (*S3Storage, error) {

	s3Storage := &S3Storage{
		bucket: c.Bucket,
	}
	// Create a new session
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
	s3Storage.session = sess

	return s3Storage, nil
}

func loadEnv(file string) {
	slog.Info("Loading environment variables", "file", file)
	// Load environment variables from .env file
	err := godotenv.Load(file)
	if err != nil {
		slog.Error("Error loading environment variable", "file", file, "error", err)
	}
}
func bucketExists(s3Client *s3.S3, bucket string) (bool, error) {
	_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		// If the error is a 404, the bucket doesn't exist
		var aErr awserr.RequestFailure
		if errors.As(err, &aErr) {
			if aErr.StatusCode() == http.StatusNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}
