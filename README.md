# fluentbit-sink

This program provides an HTTP server that can ingest JSON-formatted records from
fluent-bit and then write them to disk. It's not meant for production usage, but
for debugging in situations where a full ELK stack is not needed or CPU/memory
restrictions are too tight for such a stack.

## Usage

    $ make
    $ ./fluentbit-sink [-target=records] [-pattern=%date%.json]

## License

MIT
