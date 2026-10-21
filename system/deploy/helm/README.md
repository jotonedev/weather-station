# Weather Station Helm Chart

Production Helm chart for the weather-station platform.

## Layout

```text
system/deploy/helm/weather-station/
├── Chart.yaml          # metadata (version 0.1.0)
├── values.yaml         # all configuration
└── templates/
    ├── _helpers.tpl    # names, labels, placement, annotations, keel
    ├── NOTES.txt
    ├── service.yaml    # data-ingest, database, frontend, inference-server Services
    ├── deployment-webapp.yaml          # data-ingest + frontend
    ├── deployment-database.yaml        # embedded DuckDB
    ├── deployment-inference-server.yaml
    ├── cronjob-inference.yaml
    ├── ingress.yaml
    ├── networkpolicy.yaml              # the 7 zero-trust policies
    ├── pvc-database.yaml
    └── pvc-model-cache.yaml
```

## Key values

The chart ships with values configured for my k8s cluster, so they may not fit every cluster. Review and adjust them as needed.

## Automatic image updates (Keel)

Deployments reference `image.tag: latest`.
Keel polls the registry every 15 minutes and rolls out the new image when the digest changes.
