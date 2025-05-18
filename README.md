# S3Safe

S3Safe is a simple and secure backup tool for S3 storage

[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/s3safe)](https://github.com/jkaninda/s3safe/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/s3safe)](https://goreportcard.com/report/github.com/jkaninda/s3safe)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/s3safe.svg)](https://pkg.go.dev/github.com/jkaninda/s3safe)
![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/jkaninda/s3safe?style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/jkaninda/s3safe?style=flat-square)

## Features
- Backup to S3 storage
- Restore from S3 storage
- Compress and decompress files

## Build

```sh
go build .
```
## Configuration
 Copy the `.env.example` file to `.env` and fill in the values.

Environment variables:

```config
AWS_REGION=us-east-1
AWS_ENDPOINT=https://s3.wasabisys.com
AWS_ACCESS_KEY_ID=
AWS_SECRET_KEY=
AWS_BUCKET=
AWS_FORCE_PATH="true"
AWS_DISABLE_SSL="false"
```

## Example

### Backup
When backing up, the `--compress` flag is used to compress the directory before uploading.
And the `--dest` flag is used to specify the destination path in the S3 bucket.

The following command will compress the directory `./backups` and upload it to the S3 bucket, path `/s3path`:
`compress` is for compressing the directory before uploading.

Without `compress`, all the files in the directory will be uploaded to the S3 bucket.

```shell
s3safe backup -p backups -d /s3path --compress
```
### Backup a single file
The following command will compress the file `./backups/file.txt` and upload it to the S3 bucket, path `/s3path`:

```shell
s3safe backup --file backups/file.txt --dest /s3path --compress
```

### Restore

When restoring, the `--decompress` flag is used to decompress the file after downloading.
And the `--dest` flag is used to specify the destination directory where the file will be restored.

The following command will download the file from the S3 bucket, path `/s3path` and decompress it to the directory `./backups`:

All the files in the s3 path will be downloaded to the directory `./backups`:

```shell
s3safe restore -p backups -d /s3path
```
### Restore a single file
The following command will download the file from the S3 bucket, path `/s3path` and decompress it to the directory `./backups`:

```shell
s3safe restore --file s3path/file.txt --dest backups
```

### Restore a single file with decompressing

The following command will download the file from the S3 bucket, path `/s3path` and decompress it to the directory `./backups`:

```shell
s3safe restore --file s3path/backups-2025-05-18_09-18-42.tar.gz --dest backups --decompress
```

### Backup using Docker

```shell
	docker run --rm --env-file .env --name s3safe -v "./backups:/backups" jkaninda/s3safe:latest backup --path backups -d /s3path --compress
```
### Restore using Docker

```shell
    docker run --rm --env-file .env --name s3safe -v "./backups:/backups" jkaninda/s3safe:latest restore --path backups -d /s3path --decompress
```