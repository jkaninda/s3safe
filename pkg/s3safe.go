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
	"strings"
	"time"
)

// BackupManager handles backup operations
type BackupManager struct {
	config    *Config
	s3Storage *S3Storage
}

// RestoreManager handles restore operations
type RestoreManager struct {
	config    *Config
	s3Storage *S3Storage
}

// Backup is the cobra command handler for backup
func Backup(cmd *cobra.Command) error {
	bm, err := NewBackupManager(cmd)
	if err != nil {
		return err
	}
	return bm.Backup()
}

// Restore is the cobra command handler for restore
func Restore(cmd *cobra.Command) error {
	rm, err := NewRestoreManager(cmd)
	if err != nil {
		return err
	}
	return rm.Restore()
}

// NewBackupManager creates a new BackupManager instance
func NewBackupManager(cmd *cobra.Command) (*BackupManager, error) {
	config := NewConfig(cmd)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	s3Storage, err := config.NewS3Storage()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 storage: %w", err)
	}

	return &BackupManager{
		config:    config,
		s3Storage: s3Storage,
	}, nil
}

// NewRestoreManager creates a new RestoreManager instance
func NewRestoreManager(cmd *cobra.Command) (*RestoreManager, error) {
	config := NewConfig(cmd)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	s3Storage, err := config.NewS3Storage()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 storage: %w", err)
	}

	// Normalize path
	if config.Path[0] == '/' {
		config.Path = config.Path[1:]
	}

	return &RestoreManager{
		config:    config,
		s3Storage: s3Storage,
	}, nil
}

// Backup performs the backup operation
func (bm *BackupManager) Backup() error {
	intro()
	slog.Info("Backing up data...")

	if bm.config.Compress {
		return bm.backupWithCompression()
	}
	return bm.backupWithoutCompression()
}

// Restore performs the restore operation
func (rm *RestoreManager) Restore() error {
	intro()
	slog.Info("Restoring data...")

	if err := rm.ensureDestinationExists(); err != nil {
		return err
	}

	if rm.config.File != "" {
		return rm.restoreSingleFile()
	}
	return rm.restoreMultipleFiles()
}

func (bm *BackupManager) backupWithCompression() error {
	outputFile := bm.generateOutputFilename()

	if err := compressDirectory(bm.config.Path, outputFile); err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}
	slog.Info("Compressed directory", "path", bm.config.Path, "dest", outputFile)

	targetPath := filepath.Join(bm.config.Dest, filepath.Base(outputFile))
	if err := bm.s3Storage.Upload(outputFile, targetPath); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	slog.Info("Backup completed successfully", "path", bm.config.Path, "dest", bm.config.Dest)
	return nil
}

func (bm *BackupManager) backupWithoutCompression() error {
	if bm.config.File != "" {
		return bm.uploadSingleFile()
	}
	return bm.uploadMultipleFiles()
}

func (bm *BackupManager) uploadSingleFile() error {
	sourcePath := filepath.Join(bm.config.Path, bm.config.File)
	targetPath := filepath.Join(bm.config.Dest, bm.config.File)
	return bm.s3Storage.Upload(sourcePath, targetPath)
}

func (bm *BackupManager) uploadMultipleFiles() error {
	files, err := ListFiles(bm.config.Path, bm.config.Recursive)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	for _, file := range files {
		if err := bm.processFileForUpload(file); err != nil {
			return err
		}
	}
	return nil
}

func (bm *BackupManager) processFileForUpload(file Item) error {
	if slices.Contains(bm.config.Exclude, filepath.Base(file.Key)) {
		slog.Warn("Ignoring file", "file", file.Key)
		return nil
	}

	if file.IsDir {
		return nil
	}

	sourcePath := filepath.Join(bm.config.Path, file.Key)
	targetPath := filepath.Join(bm.config.Dest, file.Key)
	return bm.s3Storage.Upload(sourcePath, targetPath)
}

func (bm *BackupManager) generateOutputFilename() string {
	baseName := filepath.Base(bm.config.Path)
	if !bm.config.Timestamp {
		return filepath.Join(bm.config.Path, fmt.Sprintf("%s.tar.gz", baseName))
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return filepath.Join(bm.config.Path, fmt.Sprintf("%s-%s.tar.gz", baseName, timestamp))
}
func (rm *RestoreManager) ensureDestinationExists() error {
	if _, err := os.Stat(rm.config.Dest); os.IsNotExist(err) {
		if err := os.MkdirAll(rm.config.Dest, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}
	return nil
}

func (rm *RestoreManager) restoreSingleFile() error {
	sourcePath := filepath.Join(rm.config.Path, rm.config.File)
	destPath := filepath.Join(rm.config.Dest, rm.config.File)

	if err := rm.s3Storage.Download(sourcePath, destPath, rm.config.Force); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if rm.config.Decompress && isCompressed(destPath) {
		if err := decompressDirectory(destPath, rm.config.Dest); err != nil {
			return fmt.Errorf("decompression failed: %w", err)
		}
		slog.Info("Decompressed file", "file", rm.config.File)
	}

	slog.Info("Restore completed successfully", "file", rm.config.File)
	return nil
}

func (rm *RestoreManager) restoreMultipleFiles() error {
	files, err := rm.s3Storage.List(rm.config.Path, rm.config.Recursive)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	for _, file := range files {
		if err := rm.processFileForDownload(file); err != nil {
			if rm.config.IgnoreErrors {
				slog.Warn("Ignoring error", "error", err)
				continue
			}
			return err
		}
	}

	slog.Info("Restore completed successfully", "path", rm.config.Path, "dest", rm.config.Dest)
	return nil
}

func (rm *RestoreManager) processFileForDownload(file Item) error {
	if slices.Contains(rm.config.Exclude, filepath.Base(file.Key)) {
		slog.Warn("Ignoring file", "file", file.Key)
		return nil
	}

	if file.IsDir {
		return nil
	}

	destPath := filepath.Join(rm.config.Dest, removePrefix(file.Key, rm.config.Path))
	if err := rm.s3Storage.Download(file.Key, destPath, rm.config.Force); err != nil {
		return fmt.Errorf("failed to download file %s: %w", file.Key, err)
	}

	if rm.config.Decompress && isCompressed(destPath) {
		if err := decompressDirectory(destPath, rm.config.Dest); err != nil {
			if rm.config.IgnoreErrors {
				slog.Warn("Ignoring decompression error", "error", err)
				return nil
			}
			return fmt.Errorf("failed to decompress file %s: %w", file.Key, err)
		}
		slog.Info("Decompressed file", "file", file.Key)
	}

	slog.Info("Downloaded file", "file", file.Key)
	return nil
}

// Validate is the cobra command handler for config validation
func Validate(cmd *cobra.Command) error {
	config := NewConfig(cmd)
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	fmt.Println("Config validated successfully")
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

func (s S3Storage) Download(path string, dest string, force bool) error {
	// Check if the destination path exists
	destPath := filepath.Dir(dest)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		err := os.MkdirAll(destPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}
	// Check if the file already exists
	if !force {
		if _, err := os.Stat(dest); err == nil {
			slog.Warn("File already exists, use --force to overwrite, skipping download", "file", dest)
			return nil
		}
	}
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

func (s S3Storage) List(path string, recursive bool) ([]Item, error) {
	svc := s3.New(s.session)
	files := make([]Item, 0)

	// Ensure the path ends with a slash for proper folder listing
	if path != "" && !strings.HasSuffix(path, "/") {
		path += "/"
	}

	var contToken *string
	var delimiter *string

	// Only use delimiter for non-recursive listing
	if !recursive {
		delimiter = aws.String("/")
	}

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(path),
			ContinuationToken: contToken,
		}

		if delimiter != nil {
			input.Delimiter = delimiter
		}

		resp, err := svc.ListObjectsV2(input)
		if err != nil {
			return files, fmt.Errorf("could not list items in S3 bucket %s: %w", s.bucket, err)
		}

		// Process actual files
		for _, item := range resp.Contents {
			// Skip the directory marker itself (the path with trailing slash)
			if *item.Key == path {
				continue
			}

			file := Item{
				Key:          *item.Key,
				LastModified: *item.LastModified,
				IsDir:        *item.Size == 0 && strings.HasSuffix(*item.Key, "/"),
			}

			files = append(files, file)
		}

		// Only process common prefixes (folders) in non-recursive mode
		if !recursive {
			for _, prefix := range resp.CommonPrefixes {
				files = append(files, Item{
					Key:          *prefix.Prefix,
					LastModified: time.Time{},
					IsDir:        true,
				})
			}
		}

		if !*resp.IsTruncated {
			break
		}

		contToken = resp.NextContinuationToken
	}

	// If recursive and we found folders (items ending with /), list them too
	if recursive {
		var subDirs []Item
		for _, file := range files {
			if file.IsDir {
				subFiles, err := s.List(file.Key, true)
				if err != nil {
					return files, err
				}
				subDirs = append(subDirs, subFiles...)
			}
		}
		files = append(files, subDirs...)
	}

	return files, nil
}

// ListFiles lists files in the local directory, optionally recursively.
func ListFiles(path string, recursive bool) ([]Item, error) {
	var files []Item

	err := walkDir(path, path, recursive, &files)
	if err != nil {
		return files, err
	}

	return files, nil
}

// walkDir is a recursive helper to collect items.
func walkDir(root, current string, recursive bool, files *[]Item) error {
	entries, err := os.ReadDir(current)
	if err != nil {
		return fmt.Errorf("could not read directory %q: %w", current, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(current, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("could not get file info for %q: %w", fullPath, err)
		}

		relPath, err := filepath.Rel(root, fullPath)
		if err != nil {
			return fmt.Errorf("could not determine relative path: %w", err)
		}

		*files = append(*files, Item{
			Key:          relPath,
			LastModified: info.ModTime(),
			IsDir:        info.IsDir(),
		})

		// If recursive and it's a directory, go deeper
		if recursive && info.IsDir() {
			if err := walkDir(root, fullPath, recursive, files); err != nil {
				return err
			}
		}
	}

	return nil
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
	// err = os.Remove(sourceFile)
	// if err != nil {
	//	slog.Error("error removing file", "file", sourceFile, "error", err)
	// }
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
