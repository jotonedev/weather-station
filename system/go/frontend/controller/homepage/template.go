package homepage

import "html/template"

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <style>
    :root {
      --bg: #0f172a;
      --card-bg: #1e293b;
      --border: #334155;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --accent: #38bdf8;
      --accent-dim: rgba(56, 189, 248, 0.15);
      --green: #4ade80;
      --red: #f87171;
    }
    @media (prefers-color-scheme: light) {
      :root {
        --bg: #f8fafc;
        --card-bg: #ffffff;
        --border: #e2e8f0;
        --text: #0f172a;
        --text-muted: #64748b;
        --accent: #0284c7;
        --accent-dim: rgba(2, 132, 199, 0.1);
        --green: #16a34a;
        --red: #dc2626;
      }
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 1.5rem;
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
    }
    header {
      display: flex;
      flex-wrap: wrap;
      justify-content: space-between;
      align-items: center;
      gap: 1rem;
      padding-bottom: 1.5rem;
      border-bottom: 1px solid var(--border);
      margin-bottom: 1.5rem;
    }
    .title-group h1 {
      font-size: 1.75rem;
      font-weight: 700;
      letter-spacing: -0.025em;
    }
    .station-subtitle {
      font-size: 0.875rem;
      color: var(--text-muted);
      margin-top: 0.25rem;
    }
    .station-subtitle strong {
      color: var(--accent);
      font-family: ui-monospace, monospace;
    }
    .station-selector form {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .station-selector select {
      background: var(--card-bg);
      color: var(--text);
      border: 1px solid var(--border);
      padding: 0.5rem 0.75rem;
      border-radius: 0.375rem;
      font-size: 0.875rem;
    }
    .station-selector button {
      background: var(--accent);
      color: #ffffff;
      border: none;
      padding: 0.5rem 0.875rem;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      font-weight: 500;
      cursor: pointer;
    }
    .alert-error {
      background: rgba(248, 113, 113, 0.1);
      border: 1px solid var(--red);
      color: var(--red);
      padding: 0.875rem 1rem;
      border-radius: 0.5rem;
      margin-bottom: 1.5rem;
      font-size: 0.875rem;
    }
    .timestamp-bar {
      display: flex;
      flex-wrap: wrap;
      justify-content: space-between;
      align-items: center;
      gap: 0.5rem;
      background: var(--card-bg);
      border: 1px solid var(--border);
      padding: 0.75rem 1rem;
      border-radius: 0.5rem;
      margin-bottom: 1.5rem;
      font-size: 0.875rem;
      color: var(--text-muted);
    }
    .timestamp-bar strong {
      color: var(--text);
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 1.25rem;
      margin-bottom: 2rem;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 0.75rem;
      padding: 1.25rem;
    }
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1rem;
      padding-bottom: 0.5rem;
      border-bottom: 1px solid var(--border);
    }
    .card-title {
      font-size: 0.875rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--accent);
    }
    .metrics {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }
    .metric {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
    }
    .metric-label {
      font-size: 0.875rem;
      color: var(--text-muted);
    }
    .metric-value {
      font-size: 1.125rem;
      font-weight: 600;
    }
    .metric-value.featured {
      font-size: 1.5rem;
      color: var(--accent);
    }
    .empty-state {
      text-align: center;
      padding: 3rem 1.5rem;
      background: var(--card-bg);
      border: 1px dashed var(--border);
      border-radius: 0.75rem;
      margin-bottom: 2rem;
    }
    .empty-state h3 {
      font-size: 1.125rem;
      font-weight: 600;
      margin-bottom: 0.5rem;
    }
    .empty-state p {
      color: var(--text-muted);
      font-size: 0.875rem;
    }
    .history-card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 0.75rem;
      padding: 1.25rem;
      overflow-x: auto;
    }
    .history-card h2 {
      font-size: 1rem;
      font-weight: 600;
      margin-bottom: 1rem;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.875rem;
      text-align: left;
    }
    th {
      padding: 0.625rem 0.75rem;
      border-bottom: 1px solid var(--border);
      color: var(--text-muted);
      font-weight: 600;
      font-size: 0.75rem;
      text-transform: uppercase;
    }
    td {
      padding: 0.625rem 0.75rem;
      border-bottom: 1px solid var(--border);
    }
    tr:last-child td {
      border-bottom: none;
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div class="title-group">
        <h1>Weather Station</h1>
        <div class="station-subtitle">Station: <strong>{{ .SelectedNodeID }}</strong></div>
      </div>
      {{ if .AvailableNodes }}
      <div class="station-selector">
        <form method="GET" action="/">
          <label for="node_id" class="metric-label">Switch Station:</label>
          <select id="node_id" name="node_id">
            {{ range .AvailableNodes }}
            <option value="{{ . }}" {{ if eq . $.SelectedNodeID }}selected{{ end }}>{{ . }}</option>
            {{ end }}
          </select>
          <button type="submit">View</button>
        </form>
      </div>
      {{ end }}
    </header>

    {{ if .ErrorMessage }}
    <div class="alert-error">{{ .ErrorMessage }}</div>
    {{ end }}

    {{ if .LatestReading }}
    <div class="timestamp-bar">
      <span>Latest sample taken at <strong>{{ .LatestReading.FormattedTime }}</strong></span>
      <span>Station ID: <strong>{{ .LatestReading.NodeID }}</strong></span>
    </div>

    <div class="grid">
      <!-- Climate Card -->
      <div class="card">
        <div class="card-header">
          <span class="card-title">Climate</span>
        </div>
        <div class="metrics">
          <div class="metric">
            <span class="metric-label">Temperature</span>
            <span class="metric-value featured">{{ .LatestReading.Temperature }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Humidity</span>
            <span class="metric-value">{{ .LatestReading.Humidity }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Pressure</span>
            <span class="metric-value">{{ .LatestReading.Pressure }}</span>
          </div>
        </div>
      </div>

      <!-- Wind Card -->
      <div class="card">
        <div class="card-header">
          <span class="card-title">Wind</span>
        </div>
        <div class="metrics">
          <div class="metric">
            <span class="metric-label">Wind Speed</span>
            <span class="metric-value featured">{{ .LatestReading.WindSpeed }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Direction</span>
            <span class="metric-value">{{ .LatestReading.WindAngle }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Raw Speed</span>
            <span class="metric-value">{{ .LatestReading.RawWindSpeed }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Raw Direction</span>
            <span class="metric-value">{{ .LatestReading.RawWindAngle }}</span>
          </div>
        </div>
      </div>

      <!-- Air Quality Card -->
      <div class="card">
        <div class="card-header">
          <span class="card-title">Air Quality</span>
        </div>
        <div class="metrics">
          <div class="metric">
            <span class="metric-label">PM2.5</span>
            <span class="metric-value featured">{{ .LatestReading.PM2_5 }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">PM10</span>
            <span class="metric-value">{{ .LatestReading.PM10_0 }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">PM1.0</span>
            <span class="metric-value">{{ .LatestReading.PM1_0 }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">VOC</span>
            <span class="metric-value">{{ .LatestReading.VOC }}</span>
          </div>
        </div>
      </div>

      <!-- Rain Card -->
      <div class="card">
        <div class="card-header">
          <span class="card-title">Rainfall</span>
        </div>
        <div class="metrics">
          <div class="metric">
            <span class="metric-label">Rain Rate</span>
            <span class="metric-value featured">{{ .LatestReading.RainRate }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Last 1 Hour</span>
            <span class="metric-value">{{ .LatestReading.Rain1h }}</span>
          </div>
          <div class="metric">
            <span class="metric-label">Last 24 Hours</span>
            <span class="metric-value">{{ .LatestReading.Rain24h }}</span>
          </div>
        </div>
      </div>

      <!-- Diagnostics Card -->
      <div class="card">
        <div class="card-header">
          <span class="card-title">Diagnostics</span>
        </div>
        <div class="metrics">
          <div class="metric">
            <span class="metric-label">Anemometer Status / AGC</span>
            <span class="metric-value">{{ .LatestReading.AnemoStatus }} / {{ .LatestReading.AnemoAGC }} (var {{ .LatestReading.AnemoAGCVar }})</span>
          </div>
          <div class="metric">
            <span class="metric-label">Wind Vane Status / AGC</span>
            <span class="metric-value">{{ .LatestReading.VaneStatus }} / {{ .LatestReading.VaneAGC }} (var {{ .LatestReading.VaneAGCVar }})</span>
          </div>
          <div class="metric">
            <span class="metric-label">Rain Meter Status / AGC</span>
            <span class="metric-value">{{ .LatestReading.RainStatus }} / {{ .LatestReading.RainAGC }} (var {{ .LatestReading.RainAGCVar }})</span>
          </div>
        </div>
      </div>
    </div>

    {{ if .RecentReadings }}
    <div class="history-card">
      <h2>Recent Readings</h2>
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Temp</th>
            <th>Humidity</th>
            <th>Pressure</th>
            <th>Wind</th>
            <th>PM2.5</th>
            <th>Rain Rate</th>
          </tr>
        </thead>
        <tbody>
          {{ range .RecentReadings }}
          <tr>
            <td>{{ .FormattedTime }}</td>
            <td>{{ .Temperature }}</td>
            <td>{{ .Humidity }}</td>
            <td>{{ .Pressure }}</td>
            <td>{{ .WindSpeed }} {{ if ne .WindAngle "—" }}({{ .WindAngle }}){{ end }}</td>
            <td>{{ .PM2_5 }}</td>
            <td>{{ .RainRate }}</td>
          </tr>
          {{ end }}
        </tbody>
      </table>
    </div>
    {{ end }}

    {{ else }}
    <div class="empty-state">
      <h3>No Readings Recorded Yet</h3>
      <p>Station <strong>{{ .SelectedNodeID }}</strong> has not reported any sensor readings to the database.</p>
    </div>
    {{ end }}
  </div>
</body>
</html>`

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))
