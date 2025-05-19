/*
MIT License

Copyright (c) 2025 Jonas Kaninda

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package pkg

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jkaninda/s3safe/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
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
	RetentionDays int
	Ignore        []string
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
	c.Path, _ = cmd.Flags().GetString("path")
	c.Dest, _ = cmd.Flags().GetString("dest")
	c.File, _ = cmd.Flags().GetString("file")
	c.Compress, _ = cmd.Flags().GetBool("compress")
	c.Decompress, _ = cmd.Flags().GetBool("decompress")
	c.Timestamp, _ = cmd.Flags().GetBool("timestamp")
	c.IgnoreErrors, _ = cmd.Flags().GetBool("ignore-errors")
	ignores, _ := cmd.Flags().GetString("ignore")
	c.Ignore = strings.Split(ignores, ",")
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
		return fmt.Errorf("region is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.KeyID == "" {
		return fmt.Errorf("key id is required")
	}
	if c.Secret == "" {
		return fmt.Errorf("secret is required")
	}
	if c.EndPoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	return nil
}
func (c *Config) newS3Storage() (*S3Storage, error) {
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
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
