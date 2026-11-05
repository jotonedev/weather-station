package static

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed assets
var embeddedAssets embed.FS

var (
	// ErrInvalidAssetPath is returned when an asset path fails traversal or format validation.
	ErrInvalidAssetPath = errors.New("invalid static asset path")
	// ErrAssetNotFound is returned when a requested static file does not exist.
	ErrAssetNotFound = errors.New("static asset not found")
)

// Asset represents an embedded static file with cached metadata and optional pre-compressed data.
type Asset struct {
	Data         []byte
	GzipData     []byte
	ContentType  string
	ETag         string
	CacheControl string
}

// Controller serves embedded static assets with path traversal protection.
type Controller struct {
	log    *zap.Logger
	assets map[string]Asset
}

// ControllerParams defines dependencies for Controller.
type ControllerParams struct {
	fx.In

	Logger *zap.Logger
}

// New creates a new static asset Controller instance.
func New(params ControllerParams) (*Controller, error) {
	subFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		return nil, err
	}

	assets := make(map[string]Asset)
	err = fs.WalkDir(subFS, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		ext := path.Ext(p)
		contentType, ok := contentTypeForExt(ext)
		if !ok {
			return nil
		}

		data, readErr := fs.ReadFile(subFS, p)
		if readErr != nil {
			return readErr
		}

		var gzipData []byte
		if isCompressibleExt(ext) {
			compressed, compErr := compressGzip(data)
			if compErr == nil && len(compressed) < len(data) {
				gzipData = compressed
			}
		}

		hash := sha256.Sum256(data)
		assets[p] = Asset{
			Data:         data,
			GzipData:     gzipData,
			ContentType:  contentType,
			ETag:         fmt.Sprintf("W/\"%x\"", hash),
			CacheControl: cacheControlForExt(ext),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Controller{
		log:    params.Logger,
		assets: assets,
	}, nil
}

// GetAsset retrieves a static asset by relative path.
func (c *Controller) GetAsset(rawPath string) (Asset, error) {
	if rawPath == "" || strings.Contains(rawPath, "\\") || strings.Contains(rawPath, "..") {
		return Asset{}, ErrInvalidAssetPath
	}

	trimmed := strings.TrimPrefix(rawPath, "/")
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "/" || strings.HasPrefix(cleaned, "..") || !fs.ValidPath(cleaned) {
		return Asset{}, ErrInvalidAssetPath
	}

	asset, ok := c.assets[cleaned]
	if !ok {
		return Asset{}, ErrAssetNotFound
	}

	return asset, nil
}

// ReadAsset validates the relative asset path, reads the embedded file, and returns its bytes and MIME content type.
func (c *Controller) ReadAsset(rawPath string) ([]byte, string, error) {
	asset, err := c.GetAsset(rawPath)
	if err != nil {
		return nil, "", err
	}
	return asset.Data, asset.ContentType, nil
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isCompressibleExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".css", ".ttf", ".txt":
		return true
	default:
		return false
	}
}

func contentTypeForExt(ext string) (string, bool) {
	switch strings.ToLower(ext) {
	case ".css":
		return "text/css; charset=utf-8", true
	case ".woff2":
		return "font/woff2", true
	case ".woff":
		return "font/woff", true
	case ".ttf":
		return "font/ttf", true
	case ".txt":
		return "text/plain; charset=utf-8", true
	default:
		return "", false
	}
}

func cacheControlForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".woff2", ".woff", ".ttf":
		return "public, max-age=31536000, immutable"
	default:
		return "public, max-age=0, must-revalidate"
	}
}
