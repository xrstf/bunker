# Bunker

This program provides an HTTP server that can ingest JSON-formatted records from
fluent-bit and then write them to disk. It's not meant for production usage, but
for debugging in situations where a full ELK stack is not needed or CPU/memory
restrictions are too tight for such a stack.

## Usage

    $ make
    $ ./bunker \
        [-target=records] \
        [-pattern=%date%/%kubernetes_namespace_name%.json] \
        [-listen=0.0.0.0:9095]

Pods with the annotation `xrstf.de/bunker=ignore` will have their logs ignored and
not persisted. Anything else received from fluent-bit will get written to disk.

## Metrics

Bunker exposes a Prometheus-compatible `/metrics` endpoint, providing these metrics:

* `bunker_requests_total` is the total number of handled HTTP requests (labelled with
  the resulting HTTP status code).
* `bunker_open_writers_total` is the number of currently opened file handles.
* `bunker_ingested_records_total` is the total number of ingested log records. Does
  not include excluded records.
* `bunker_received_records_total` is the total number of received log records, including
  those excluded via pod annotations.

## License

MIT
