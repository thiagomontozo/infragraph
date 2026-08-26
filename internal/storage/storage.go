package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Object struct {
	Key, ContentHash string
	Size             int64
	MediaType        string
}
type ObjectStorage interface {
	Put(context.Context, string, io.Reader, int64, string) (Object, error)
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	Ready(context.Context) error
}
type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	r, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(r, 0700); e != nil {
		return nil, e
	}
	return &Local{r}, nil
}
func (l *Local) file(key string) (string, error) {
	if key == "" || strings.Contains(key, "..") || filepath.IsAbs(key) {
		return "", errors.New("invalid object key")
	}
	p := filepath.Join(l.root, filepath.FromSlash(key))
	rel, e := filepath.Rel(l.root, p)
	if e != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("object key escapes storage root")
	}
	return p, nil
}
func (l *Local) Put(_ context.Context, key string, r io.Reader, size int64, media string) (Object, error) {
	p, e := l.file(key)
	if e != nil {
		return Object{}, e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return Object{}, e
	}
	tmp := p + ".pending"
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return Object{}, e
	}
	h := sha256.New()
	n, e := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, size+1))
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e == nil && n != size {
		e = fmt.Errorf("object size mismatch: expected %d got %d", size, n)
	}
	if e == nil {
		e = os.Rename(tmp, p)
	}
	if e != nil {
		os.Remove(tmp)
		return Object{}, e
	}
	return Object{key, hex.EncodeToString(h.Sum(nil)), n, media}, nil
}
func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	p, e := l.file(key)
	if e != nil {
		return nil, e
	}
	return os.Open(p)
}
func (l *Local) Delete(_ context.Context, key string) error {
	p, e := l.file(key)
	if e != nil {
		return e
	}
	e = os.Remove(p)
	if os.IsNotExist(e) {
		return nil
	}
	return e
}
func (l *Local) Ready(_ context.Context) error {
	f, e := os.CreateTemp(l.root, "ready-")
	if e != nil {
		return e
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(endpoint, bucket, access, secret string, tls bool) (*S3, error) {
	if endpoint == "" || bucket == "" || access == "" || secret == "" {
		return nil, errors.New("complete S3 configuration required")
	}
	c, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(access, secret, ""), Secure: tls})
	if e != nil {
		return nil, e
	}
	return &S3{c, bucket}, nil
}
func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, media string) (Object, error) {
	h := sha256.New()
	info, e := s.client.PutObject(ctx, s.bucket, key, io.TeeReader(r, h), size, minio.PutObjectOptions{ContentType: media})
	if e != nil {
		return Object{}, e
	}
	return Object{key, hex.EncodeToString(h.Sum(nil)), info.Size, media}, nil
}
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	o, e := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if e != nil {
		return nil, e
	}
	if _, e = o.Stat(); e != nil {
		o.Close()
		return nil, e
	}
	return o, nil
}
func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
func (s *S3) Ready(ctx context.Context) error {
	exists, e := s.client.BucketExists(ctx, s.bucket)
	if e != nil {
		return e
	}
	if !exists {
		return errors.New("S3 bucket does not exist")
	}
	return nil
}
