package main

import (
    "fmt"
	"net/url"
)

type FileURI struct {
    Scheme  string
    Bucket  string
    Path    string
}

func FileURINew(path string) (*FileURI, error) {
    u, err := url.Parse(path)
    if err != nil {
        return nil, err
    }
    if u.Scheme != "" && u.Scheme != "s3" && u.Scheme != "file" {
        return nil, fmt.Errorf("Invalid URI scheme must be one of file/s3/NONE")
    }

    uri := FileURI{
        Scheme: u.Scheme,
        Bucket: u.Host,
        Path: u.Path,
    }

    if uri.Scheme == "" {
        uri.Scheme = "file"
    }
    if uri.Scheme == "s3" {
        uri.Path = uri.Path[1:]
    }
    if uri.Path == "" && uri.Scheme == "s3" {
        uri.Path = "/"
    }

    return &uri, nil
}

// Return the path as a valid S3 bucket key
func (uri *FileURI) Key() *string {
    if uri.Path[0] == '/' {
        s := uri.Path[1:]
        return &s
    }
    return &uri.Path
}

// Return a string version of the path
func (uri *FileURI) String() string {
    if uri.Scheme == "s3" {
        return fmt.Sprintf("s3://%s/%s", uri.Bucket, uri.Key())
    } else {
        return fmt.Sprintf("file://%s", uri.Path)
    }
}