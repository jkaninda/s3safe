# S3Safe - Secure S3 Backup & Restore Tool

[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/s3safe)](https://github.com/jkaninda/s3safe/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/s3safe)](https://goreportcard.com/report/github.com/jkaninda/s3safe)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/s3safe.svg)](https://pkg.go.dev/github.com/jkaninda/s3safe)
![Docker Image Size](https://img.shields.io/docker/image-size/jkaninda/s3safe?style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/jkaninda/s3safe?style=flat-square)

S3Safe is a lightweight CLI tool for securely backing up and restoring data from Amazon S3 and S3-compatible storage.

## Key Features
- **Secure transfers** to/from S3-compatible storage
- **Compression support** (gzip/tar)
- **Flexible operations**:
    - Backup entire directories or single files
    - Restore with optional decompression
    - Recursive operations
    - Exclusion patterns
- **Docker support** for containerized environments

## Installation
```shell
# Using Go
go install github.com/jkaninda/s3safe@latest

# Using Docker
docker pull jkaninda/s3safe:latest
```

## Configuration
Copy `.env.example` to `.env` and configure your S3 credentials:

```ini
AWS_REGION=us-east-1
AWS_ENDPOINT=https://s3.wasabisys.com
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_KEY=your_secret_key
AWS_BUCKET=your_bucket_name
AWS_FORCE_PATH="true"  # For path-style URLs
AWS_DISABLE_SSL="false"  # Set "true" for non-HTTPS endpoints
```

## Command Reference

### Global Options
| Option            | Short | Description                                          |
|-------------------|-------|------------------------------------------------------|
| `--exclude`       | `-e`  | Exclude files/directories (comma-separated patterns) |
| `--recursive`     | `-r`  | Process directories recursively                      |
| `--path`          | `-p`  | Source directory path                                |
| `--dest`          | `-d`  | Destination path (in S3 or local filesystem)         |
| `--file`          | `-f`  | Process single file instead of directory             |
| `--ignore-errors` | `-i`  | Continue on errors during restore                    |
| `--env-file`      |       | Custom environment file (default: .env)              |
| `--bucket`        | `b`   | S3 bucket name                                       |
| `--help`          | `-h`  | Show help message                                    |
| `--version`       | `-v`  | Show version information                             |

### Backup Options
| Option          | Short | Description                                |
|-----------------|-------|--------------------------------------------|
| `--compress`    | `-c`  | Compress before upload (creates .tar.gz)   |
| `--timestamp`   | `-t`  | Add timestamp to compressed filename       |

### Restore Options
| Option         | Short | Description                                                 |
|----------------|-------|-------------------------------------------------------------|
| `--decompress` | `-D`  | Decompress after download                                   |
| `--force`      |       | Force restore to destination path, overwrite existing files |

## Usage Examples

### Backup Operations

**Backup directory (compressed):**
```shell
s3safe backup -p ./backups -d /s3path --compress --timestamp
```

**Backup single file:**

```shell
s3safe backup --file data.db --dest /s3path/db-backups --compress
```

**Non-compressed directory backup:**
```shell
s3safe backup -p ./backups -d /s3path/backups -r
```

### Restore Operations
**Restore compressed backup:**
```shell
s3safe restore -p /s3path/backup.tar.gz -d ./backups --decompress
```

**Restore directory (recursive):**

```shell
s3safe restore --path /s3path --dest ./backups --recursive
```

### Docker Usage
**Backup with Docker:**
```shell
docker run --rm --env-file .env \
  -v "./backups:/backups" \
  jkaninda/s3safe:latest \
  backup --path /backups -d s3path --compress
```

**Restore with Docker:**
```shell
docker run --rm --env-file .env \
  -v "./restored:/restored" \
  jkaninda/s3safe:latest \
  restore --path s3path/backup.tar.gz -d /restored --decompress
```

## License
MIT License - See [LICENSE](LICENSE) for details.

## Contributing
Contributions welcome! Please open an issue or PR on [GitHub](https://github.com/jkaninda/s3safe)


---
### Copyright (c) 2025 Jonas Kaninda

