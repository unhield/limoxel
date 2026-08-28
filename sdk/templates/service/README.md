# Limoxel SDK HTTP Microservice Template

A lightweight HTTP REST microservice template exposing Limoxel SDK intelligence capabilities over JSON endpoints.

## Endpoints

- `GET /api/v1/info`: Repository identity, languages, and capabilities.
- `GET /api/v1/health`: Multidimensional repository health evaluation.
- `GET /api/v1/search?q={query}`: Symbol and entity search.

## Running the Service

```bash
go run server.go
```
