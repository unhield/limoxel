# Limoxel SDK CI/CD Integration Template

An automated Quality Gate script designed for continuous integration workflows (e.g. GitHub Actions, GitLab CI) that checks repository health scores and enforces boundary compliance.

## Usage in CI Pipeline

```bash
# Run quality gate with minimum required score of 70
go run check_health.go -repo=. -min-score=70.0
```

## Exit Codes

- `0`: Quality gate passed.
- `1`: Health score below threshold or critical analysis error.
