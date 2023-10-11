# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2023-10-11

### Added

- New histogram metrics. Every existing summary Prometheus metric now has a corresponding
  histogram metric, with the `_hist_` in the metrics name.

### Removed

- "io/ioutil" library which is deprecated since go-1.16.

## [0.4.3] - 2023-02-28

### Added

- Switch to Go 1.20 and also update prometheus `client_golang` library to 1.13.1

## [0.4.2] - 2022-06-01

### Added

- Closed bug #28

## [0.4.1] - 2022-02-01

### Added

- New CLI arg `--ab-time-per-request` show Apache Benchmark style time per request
  metric, which is hidden by default. Fixes #12.

### Removed

- `OverallRequestTimeSeconds` has been removed from the JSON report to avoid confusion.
   Related to #12.

## [0.4.0] - 2021-12-07

### Added

- Initial public release.
