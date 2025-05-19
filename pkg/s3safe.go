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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	goutils "github.com/jkaninda/go-utils"
	"github.com/jkaninda/s3safe/utils"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func Backup(cmd *cobra.Command) error {
	intro()
	slog.Info("Backing up data...")
	// New config
	c := &Config{}
	c.NewConfig(cmd)
	// New S3 storage
	s3Storage, err := c.newS3Storage()
	if err != nil {
		return fmt.Errorf("failed to create S3 storage: %w", err)
	}
	if c.Compress {
		outPut := filepath.Join(c.Path, fmt.Sprintf("%s.tar.gz", filepath.Base(c.Path)))
		if c.Timestamp {
			currentTime := time.Now()
			// Format the timestamp
			timestamp := currentTime.Format("2006-01-02_15-04-05")
			outPut = filepath.Join(c.Path, fmt.Sprintf("%s-%s.tar.gz", filepath.Base(c.Path), timestamp))
		}
		// Compress the file
		err = compressDirectory(c.Path, outPut)
		if err != nil {
			return fmt.Errorf("failed to compress directory: %w", err)
		}
		slog.Info("Compressed directory", "path", c.Path, "dest", outPut)
		// Upload the file
		err = s3Storage.Upload(outPut, filepath.Join(c.Dest, filepath.Base(outPut)))
		if err != nil {
			return fmt.Errorf("failed to upload file: %w", err)
		}
	} else {
		// Check if is a single file
		if c.File != "" {
			// Upload the file
			err = s3Storage.Upload(filepath.Join(c.Path, c.File), filepath.Join(c.Dest, c.File))
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}
			return nil
		}
		// List the files
		files, err := ListFile(c.Path)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}
		// Upload the files
		for _, file := range files {
			if slices.Contains(c.Exclude, file.Key) {
				slog.Info("Ignoring file", "file", file.Key)
				continue
			}
			if file.IsDir {
				// check if the compression is enabled
				continue
			}
			err = s3Storage.Upload(filepath.Join(c.Path, file.Key), filepath.Join(c.Dest, file.Key))
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

		}
	}
	slog.Info("Backup completed successfully, ", "path", c.Path, "dest", c.Dest)
	return nil
}
func Restore(cmd *cobra.Command) error {
	intro()
	slog.Info("Restoring data...")
	// New config
	c := &Config{}
	c.NewConfig(cmd)
	// New S3 storage
	s3Storage, err := c.newS3Storage()
	if err != nil {
		return fmt.Errorf("failed to create S3 storage: %w", err)
	}
	// Check if the path starts with /
	if c.Path[0] == '/' {
		c.Path = c.Path[1:]
	}
	// Check if the destination path exists
	if _, err := os.Stat(c.Dest); os.IsNotExist(err) {
		err := os.MkdirAll(c.Dest, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}
	// Check if is a single file
	if c.File != "" {
		// Download the file
		err = s3Storage.Download(filepath.Join(c.Path, c.File), filepath.Join(c.Dest, c.File))
		if err != nil {
			return fmt.Errorf("failed to download file: %w", err)
		}
		// Check if the file is compressed and decompress it
		if c.Decompress && isCompressed(filepath.Join(c.Dest, c.File)) {
			err = decompressDirectory(filepath.Join(c.Dest, c.File), c.Dest)
			if err != nil {
				return fmt.Errorf("failed to decompress file: %w", err)
			}
			slog.Info("Decompressed file", "file", c.File)
		}

		slog.Info("Restore completed successfully", "file", c.File)
		return nil
	}
	// List the files
	files, err := s3Storage.List(c.Path)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}
	// Download the files
	for _, file := range files {
		if slices.Contains(c.Exclude, removePrefix(file.Key, c.Path)) {
			slog.Info("Ignoring file", "file", removePrefix(file.Key, c.Path))
			continue
		}
		err = s3Storage.Download(file.Key, filepath.Join(c.Dest, removePrefix(file.Key, c.Path)))
		if err != nil {
			if c.IgnoreErrors {
				slog.Error("Ignoring error", "error", err)
				continue
			}
			return fmt.Errorf("failed to download file: %w", err)
		}
		// Check if the file is compressed and decompress it
		if c.Decompress && isCompressed(filepath.Join(c.Dest, removePrefix(file.Key, c.Path))) {
			err = decompressDirectory(filepath.Join(c.Dest, removePrefix(file.Key, c.Path)), c.Dest)
			if err != nil {
				if c.IgnoreErrors {
					slog.Error("Ignoring error", "error", err)
					continue
				}
				return fmt.Errorf("failed to decompress file: %w", err)
			}
			slog.Info("Decompressed file", "file", file.Key)
		}
		slog.Info("Downloaded file", "file", file.Key)
	}
	slog.Info("Restore completed successfully", "path", c.Path, "dest", c.Dest)
	return nil
}
func (s S3Storage) Upload(path string, target string) error {

	// Check if file exists
	if !goutils.FileExists(path) {
		return fmt.Errorf("file %s does not exist", path)

	}
	slog.Info("Uploading file", "file", path, "size", utils.FileSize(path), "target", target)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("upload error: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			slog.Error("error closing file,", "error", err)
		}
	}(file)

	uploader := s3manager.NewUploader(s.session)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(target),
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("unable to upload %q to %q: %w", path, s.bucket, err)
	}
	slog.Info("Upload completed successfully", "file", path, "target", target)
	return nil
}

func (s S3Storage) Download(path string, dest string) error {
	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}(file)

	downloader := s3manager.NewDownloader(s.session)

	_, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return fmt.Errorf("unable to download %q from %q: %w", path, s.bucket, err)
	}

	return nil
}

func (s S3Storage) List(path string) ([]Item, error) {
	svc := s3.New(s.session)

	files := make([]Item, 0)

	var contToken *string

	for {
		resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(path),
			ContinuationToken: contToken,
		})

		if err != nil {
			return files, fmt.Errorf("could not list items in S3 bucket %s: %w", s.bucket, err)
		}

		for _, item := range resp.Contents {
			file := Item{
				Key:          *item.Key,
				LastModified: *item.LastModified,
			}

			files = append(files, file)
		}

		if !*resp.IsTruncated {
			break
		}

		contToken = resp.NextContinuationToken
	}

	return files, nil
}

// ListFile lists files in the local directory
func ListFile(path string) ([]Item, error) {
	files := make([]Item, 0)

	dir, err := os.Open(path)
	if err != nil {
		return files, fmt.Errorf("could not open directory: %w", err)
	}
	defer func(dir *os.File) {
		err := dir.Close()
		if err != nil {
			fmt.Printf("error closing directory: %v\n", err)
		}
	}(dir)

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return files, fmt.Errorf("could not read directory: %w", err)
	}

	for _, fileInfo := range fileInfos {
		file := Item{
			Key:          fileInfo.Name(),
			LastModified: fileInfo.ModTime(),
			IsDir:        fileInfo.IsDir(),
		}
		files = append(files, file)
	}

	return files, nil
}

// compressDirectory compresses a directory into a tar.gz file
func compressDirectory(sourceDir, outputFile string) error {
	slog.Info("Compressing directory", "sourceDir", sourceDir, "outputFile", outputFile)
	absOutputFile, err := filepath.Abs(outputFile)
	if err != nil {
		return fmt.Errorf("could not get absolute path of output file: %w", err)
	}

	outFile, err := os.Create(absOutputFile)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {
			slog.Error("error closing output file", "error", err)
		}
	}(outFile)

	gw := gzip.NewWriter(outFile)
	defer func(gw *gzip.Writer) {
		err := gw.Close()
		if err != nil {
			slog.Error("error closing gzip writer", "error", err)
		}
	}(gw)

	tw := tar.NewWriter(gw)
	defer func(tw *tar.Writer) {
		err := tw.Close()
		if err != nil {
			slog.Error("error closing tar writer", "error", err)
		}
	}(tw)

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the output file
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if absPath == absOutputFile {
			return nil
		}

		// Skip directories, tar only needs file headers
		if info.IsDir() {
			return nil
		}

		// Get path relative to the sourceDir
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				slog.Info("error closing file", "error", err)
			}
		}(file)

		// Create header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content
		if _, err := io.Copy(tw, file); err != nil {
			return err
		}

		return nil
	})
}

// decompressDirectory decompresses a tar.gz file into a directory
func decompressDirectory(sourceFile, destDir string) error {
	// Open the tar.gz file
	file, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}(file)

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer func(gzr *gzip.Reader) {
		err := gzr.Close()
		if err != nil {
			slog.Error("error closing gzip reader", "error", err)
		}
	}(gzr)

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read tar header: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("could not create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("could not create file: %w", err)
			}
			defer func(outFile *os.File) {
				err := outFile.Close()
				if err != nil {
					slog.Error("error closing output file", "error", err)
				}
			}(outFile)

			if _, err := io.Copy(outFile, tr); err != nil {
				return fmt.Errorf("could not write to file: %w", err)
			}
		default:
			return fmt.Errorf("unsupported type: %c in %s", header.Typeflag, header.Name)
		}
	}
	// Delete the original file
	err = os.Remove(sourceFile)
	if err != nil {
		slog.Error("error removing file", "file", sourceFile, "error", err)
	}
	return nil
}

// Check if the file is compressed
func isCompressed(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}(file)

	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return false
	}

	return string(buf[:2]) == "\x1f\x8b"
}

// // Check if file has relative path
func isRelativePath(filePath string) bool {
	return !filepath.IsAbs(filePath)
}

// IsAbsolutePath checks if a given path is absolute.
func IsAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// removePrefix removes the prefix from the file path
func removePrefix(filePath, prefix string) string {
	if len(filePath) < len(prefix) {
		return filePath
	}
	if filePath[:len(prefix)] == prefix {
		return filePath[len(prefix):]
	}
	return filePath
}

// intro prints the intro message
func intro() {
	fmt.Printf("Version: %s\n", utils.Version)
	fmt.Println("Copyright (c) 2025 Jonas Kaninda")
}
