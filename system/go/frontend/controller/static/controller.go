package static

import (
	"embed"
	"errors"
	"io"
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

// Controller serves embedded static assets with path traversal protection.
type Controller struct {
	log     *zap.Logger
	assetFS fs.FS
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
	return &Controller{
		log:     params.Logger,
		assetFS: subFS,
	}, nil
}

// ReadAsset validates the relative asset path, reads the embedded file, and returns its bytes and MIME content type.
func (c *Controller) ReadAsset(rawPath string) ([]byte, string, error) {
	if rawPath == "" || strings.Contains(rawPath, "\\") || strings.Contains(rawPath, "..") {
		return nil, "", ErrInvalidAssetPath
	}

	trimmed := strings.TrimPrefix(rawPath, "/")
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "/" || strings.HasPrefix(cleaned, "..") || !fs.ValidPath(cleaned) {
		return nil, "", ErrInvalidAssetPath
	}

	contentType, ok := contentTypeForExt(path.Ext(cleaned))
	if !ok {
		return nil, "", ErrAssetNotFound
	}

	file, err := c.assetFS.Open(cleaned)
	if err != nil {
		return nil, "", ErrAssetNotFound
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return nil, "", ErrAssetNotFound
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}

	return data, contentType, nil
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
