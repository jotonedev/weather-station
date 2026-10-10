package weatherstation

import (
	"encoding/json"
	"time"
)

// Reading represents a raw weather station reading parsed from an incoming request.
type Reading struct {
	NodeID           string    `json:"node_id" yaml:"node_id"`
	Timestamp        time.Time `json:"timestamp" yaml:"timestamp"`
	WindSpeed        *float32  `json:"wind_speed,omitempty" yaml:"wind_speed,omitempty"`
	WindAngle        *float32  `json:"wind_angle,omitempty" yaml:"wind_angle,omitempty"`
	RawWindSpeed     *float32  `json:"raw_wind_speed,omitempty" yaml:"raw_wind_speed,omitempty"`
	RawWindAngle     *float32  `json:"raw_wind_angle,omitempty" yaml:"raw_wind_angle,omitempty"`
	Temperature      *float32  `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	Humidity         *float32  `json:"humidity,omitempty" yaml:"humidity,omitempty"`
	Pressure         *float32  `json:"pressure,omitempty" yaml:"pressure,omitempty"`
	Rain1h           *float32  `json:"rain_1h,omitempty" yaml:"rain_1h,omitempty"`
	Rain24h          *float32  `json:"rain_24h,omitempty" yaml:"rain_24h,omitempty"`
	RainRate         *float32  `json:"rain_rate,omitempty" yaml:"rain_rate,omitempty"`
	PM1_0            *int32    `json:"pm1_0,omitempty" yaml:"pm1_0,omitempty"`
	PM2_5            *int32    `json:"pm2_5,omitempty" yaml:"pm2_5,omitempty"`
	PM10_0           *int32    `json:"pm10_0,omitempty" yaml:"pm10_0,omitempty"`
	VOC              *int32    `json:"voc,omitempty" yaml:"voc,omitempty"`
	AnemoStatus      *int32    `json:"anemo_status,omitempty" yaml:"anemo_status,omitempty"`
	AnemoAGC         *int32    `json:"anemo_agc,omitempty" yaml:"anemo_agc,omitempty"`
	AnemoAGCVariance *int32    `json:"anemo_agc_variance,omitempty" yaml:"anemo_agc_variance,omitempty"`
	VaneStatus       *int32    `json:"vane_status,omitempty" yaml:"vane_status,omitempty"`
	VaneAGC          *int32    `json:"vane_agc,omitempty" yaml:"vane_agc,omitempty"`
	VaneAGCVariance  *int32    `json:"vane_agc_variance,omitempty" yaml:"vane_agc_variance,omitempty"`
	RainStatus       *int32    `json:"rain_status,omitempty" yaml:"rain_status,omitempty"`
	RainAGC          *int32    `json:"rain_agc,omitempty" yaml:"rain_agc,omitempty"`
	RainAGCVariance  *int32    `json:"rain_agc_variance,omitempty" yaml:"rain_agc_variance,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler to support aliases for fields.
func (r *Reading) UnmarshalJSON(data []byte) error {
	type Alias Reading
	aux := struct {
		*Alias
		AltTS      *time.Time `json:"ts"`
		AltRain1h  *float32   `json:"rain1h"`
		AltRain24h *float32   `json:"rain24h"`
		AltPM1     *int32     `json:"pm1"`
		AltPM25    *int32     `json:"pm25"`
		AltPM10    *int32     `json:"pm10"`
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if r.Timestamp.IsZero() && aux.AltTS != nil {
		r.Timestamp = *aux.AltTS
	}
	if r.Rain1h == nil && aux.AltRain1h != nil {
		r.Rain1h = aux.AltRain1h
	}
	if r.Rain24h == nil && aux.AltRain24h != nil {
		r.Rain24h = aux.AltRain24h
	}
	if r.PM1_0 == nil && aux.AltPM1 != nil {
		r.PM1_0 = aux.AltPM1
	}
	if r.PM2_5 == nil && aux.AltPM25 != nil {
		r.PM2_5 = aux.AltPM25
	}
	if r.PM10_0 == nil && aux.AltPM10 != nil {
		r.PM10_0 = aux.AltPM10
	}
	return nil
}
