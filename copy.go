package main

import (
	"os"
	"fmt"
	"path"
	"strings"
	"path/filepath"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Given a SRC and DST URL - copy the file
//  this is a useful helper
func copyFile(config *Config, src, dst *FileURI, ensure_directory bool) error {
    if config.Verbose {
        fmt.Printf("Copy %s -> %s\n", src.String(), dst.String())
    }
    if config.DryRun {
        return nil
    }

    switch src.Scheme + "->" + dst.Scheme {
    case "file->file":
        return fmt.Errorf("cp should not be doing local files")
    case "s3->s3":
        return copyOnS3(config, src, dst)
    case "s3->file":
        return copyToLocal(config, src, dst, ensure_directory)
    case "file->s3":
        return copyToS3(config, src, dst)
    }
    return nil
}

// Copy from S3 to local file
func copyToLocal(config *Config, src, dst *FileURI, ensure_directory bool) error {
    svc := SessionForBucket(SessionNew(config), src.Bucket)
    downloader := s3manager.NewDownloaderWithClient(svc)

    params := &s3.GetObjectInput{
        Bucket: aws.String(src.Bucket),
        Key:    src.Key(),
    }

    dst_path := dst.Path

    // if the destination is a directory then copy to a file in the directory
    sinfo, err := os.Stat(dst_path)
    if err == nil && sinfo.IsDir() {
        dst_path = path.Join(dst_path, filepath.Base(src.Path))
    }

    if ensure_directory {
    }

    fd, err := os.Create(dst_path)
    if err != nil {
        fmt.Println(err)
        return err
    }
    defer fd.Close()

    _, err = downloader.Download(fd, params)
    if err != nil {
        return err
    }

    return nil
}

// Copy from local file to S3
func copyToS3(config *Config, src, dst *FileURI) error {
    svc := SessionForBucket(SessionNew(config), dst.Bucket)
    uploader := s3manager.NewUploaderWithClient(svc)

    fd, err := os.Open(src.Path)
    if err != nil {
        return err
    }
    defer fd.Close()

    params := &s3manager.UploadInput{
        Bucket:     aws.String(dst.Bucket), // Required
        Key:        cleanBucketDestPath(src.Path, dst.Path),
        Body:       fd,
    }

    _, err = uploader.Upload(params)
    if err != nil {
        return err
    }

    return nil
}

// Copy from S3 to S3
//  -- if src and dst are the same it effects a "touch"
func copyOnS3(config *Config, src, dst *FileURI) error {
    svc := SessionForBucket(SessionNew(config), dst.Bucket)

    if strings.HasSuffix(src.Path, "/") {
        return fmt.Errorf("Invalid source for bucket to bucket copy path ends in '/'")
    }

    params := &s3.CopyObjectInput{
        Bucket:         aws.String(dst.Bucket),
        CopySource:     aws.String(fmt.Sprintf("/%s/%s", src.Bucket, src.Path[1:])),
        Key:            cleanBucketDestPath(src.Path, dst.Path),
    }

    // if this is an overwrite - note that
    if src.Bucket == dst.Bucket && *params.CopySource == fmt.Sprintf("/%s/%s", dst.Bucket, *params.Key) {
        params.MetadataDirective = aws.String("REPLACE")
    }

    _, err := svc.CopyObject(params)
    if err != nil {
        return err
    }

    return nil
}

// Take a src and dst and make a valid destination path for the bucket
//  if the dst ends in "/" add the basename of the source to the object
//  make sure the leading "/" is stripped off
func cleanBucketDestPath(src, dst string) *string {
    if strings.HasSuffix(dst, "/") {
        dst += filepath.Base(src)
    }
    if strings.HasPrefix(dst, "/") {
        dst = dst[1:]
    }
    return &dst
}