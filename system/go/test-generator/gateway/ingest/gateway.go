package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	// ErrNilHTTPClient is returned when no HTTP client is supplied.
	ErrNilHTTPClient = errors.New("http client is required")
	// ErrUnexpectedStatus is returned when data-ingest responds with a non-2xx HTTP status.
	ErrUnexpectedStatus = errors.New("unexpected http status from ingest")
)

// ReadingPayload represents a synthetic sensor reading sent to data-ingest.
type ReadingPayload struct {
	NodeID           string    `json:"node_id" yaml:"node_id"`
	Timestamp        time.Time `json:"timestamp" yaml:"timestamp"`
	WindSpeed        float32   `json:"wind_speed" yaml:"wind_speed"`
	WindAngle        float32   `json:"wind_angle" yaml:"wind_angle"`
	RawWindSpeed     float32   `json:"raw_wind_speed" yaml:"raw_wind_speed"`
	RawWindAngle     float32   `json:"raw_wind_angle" yaml:"raw_wind_angle"`
	Temperature      float32   `json:"temperature" yaml:"temperature"`
	Humidity         float32   `json:"humidity" yaml:"humidity"`
	Pressure         float32   `json:"pressure" yaml:"pressure"`
	Rain1h           float32   `json:"rain_1h" yaml:"rain_1h"`
	Rain24h          float32   `json:"rain_24h" yaml:"rain_24h"`
	RainRate         float32   `json:"rain_rate" yaml:"rain_rate"`
	PM1_0            int32     `json:"pm1_0" yaml:"pm1_0"`
	PM2_5            int32     `json:"pm2_5" yaml:"pm2_5"`
	PM10_0           int32     `json:"pm10_0" yaml:"pm10_0"`
	VOC              int32     `json:"voc" yaml:"voc"`
	AnemoStatus      int32     `json:"anemo_status" yaml:"anemo_status"`
	AnemoAGC         int32     `json:"anemo_agc" yaml:"anemo_agc"`
	AnemoAGCVariance int32     `json:"anemo_agc_variance" yaml:"anemo_agc_variance"`
	VaneStatus       int32     `json:"vane_status" yaml:"vane_status"`
	VaneAGC          int32     `json:"vane_agc" yaml:"vane_agc"`
	VaneAGCVariance  int32     `json:"vane_agc_variance" yaml:"vane_agc_variance"`
	RainStatus       int32     `json:"rain_status" yaml:"rain_status"`
	RainAGC          int32     `json:"rain_agc" yaml:"rain_agc"`
	RainAGCVariance  int32     `json:"rain_agc_variance" yaml:"rain_agc_variance"`
}

// batchRequest wraps a slice of readings for the /readings endpoint.
type batchRequest struct {
	Readings []ReadingPayload `json:"readings" yaml:"readings"`
}

// Gateway sends generated weather readings to the data-ingest HTTP service.
type Gateway struct {
	log    *zap.Logger
	client *http.Client
	cfg    Config
}

// GatewayParams defines dependencies for Gateway.
type GatewayParams struct {
	fx.In

	Logger *zap.Logger
	Client *http.Client
	Config Config
}

// New creates a new ingest Gateway instance.
func New(params GatewayParams) (*Gateway, error) {
	if params.Client == nil {
		return nil, ErrNilHTTPClient
	}
	return &Gateway{
		log:    params.Logger,
		client: params.Client,
		cfg:    params.Config,
	}, nil
}

// SendReadings posts a batch of readings to the configured data-ingest endpoint.
func (g *Gateway) SendReadings(ctx context.Context, readings []ReadingPayload) error {
	if len(readings) == 0 {
		return nil
	}

	body, err := json.Marshal(batchRequest{Readings: readings})
	if err != nil {
		return fmt.Errorf("marshal readings batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create ingest request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if g.cfg.Username != "" || g.cfg.Password != "" {
		req.SetBasicAuth(g.cfg.Username, g.cfg.Password)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("post readings to ingest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		respText, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%w %d %s", ErrUnexpectedStatus, resp.StatusCode, string(bytes.TrimSpace(respText)))
	}

	return nil
}
