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

package utils

import (
	goutils "github.com/jkaninda/go-utils"
	"os"
)

var (
	Version   = "dev"                  // appVersion is the version of the application
	BuildTime = "2025-05-18T12:00:00Z" // appBuildTime is the build time of the application
)

const (
	appName    = "s3safe"
	AppExample = `
		Backup: s3safe backup --path /path/to/backup --dest /path/to/dest,
		Restore: s3safe restore --path /s3path --dest /path/to/dest"`

	AppDescription = "A simple and secure backup tool for S3 storage" // appDesc is the description of the application)
	BackupExample  = `
		Backup: "s3safe backup --path /path/to/backup --dest /path/to/dest",
		Backup a single file: "s3safe backup --file /path/to/file --dest /path/to/dest",
		Backup with compression: "s3safe backup --path /path/to/backup --dest /path/to/dest --compress",
		Backup with timestamp: "s3safe backup --path /path/to/backup --dest /path/to/dest --compress --timestamp"`
	RestoreExample = `
		Restore: "s3safe restore --path /s3path --file backup.tar.gz --dest /path/to/dest",
		Restore a single file with decompression: "s3safe restore --path /s3path --file backup.tar.gz --dest /path/to/dest --decompress",`
	RegionEnv        = "AWS_REGION"
	KeyIDEnv         = "AWS_ACCESS_KEY_ID"
	SecretEnv        = "AWS_SECRET_KEY"
	EndPointEnv      = "AWS_ENDPOINT"
	BucketEnv        = "AWS_BUCKET"
	ForcePathEnv     = "AWS_FORCE_PATH"
	DisableSSLEnv    = "AWS_DISABLE_SSL"
	RetentionDaysEnv = "AWS_RETENTION_DAYS"
)

func Env(key string) string {
	return os.Getenv(key)
}
func BoolEnv(key string) bool {
	val := os.Getenv(key)
	if val == "" {
		return false
	}
	if val == "true" {
		return true
	}
	if val == "false" {
		return false
	}
	return false
}
func FileSize(path string) string {
	file, err := os.Stat(path)
	if err != nil {
		return goutils.ConvertBytes(0)
	}
	return goutils.ConvertBytes(uint64(file.Size()))
}
