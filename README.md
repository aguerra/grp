# grp
[![Build Status](https://travis-ci.org/aguerra/grp.svg)](https://travis-ci.org/aguerra/grp)

Grp is a extremely simple radsec proxy written in Golang: it only supports
forwarding packets incoming from the TLS 1.2 connection to a radius UDP backend.
It was my first serious Golang program (needs a rewrite...) but the version
0.1.0 is stable and battle tested serving hundreds of wifi hotspots.
It is well suited for deployment on containers as it keeps itself in the
foreground, logs to stdout and is configured only through environment
variables:

- GRP_PORT: port to listen. Default: 2083.

- GRP_CA_FILE: CA certificate file. Default: ca.crt.

- GRP_CERT_FILE: server certificate file. Default: server.crt.

- GRP_KEY_FILE: certificate key file. Default: server.key.

- GRP_RADIUS_HOST: radius backend hostname. Default: localhost.

- GRP_RADIUS_PORT: radius backend port. Default: 1812.

- GRP_RADIUS_ACCT_HOST: radius accounting backend hostname. Default: localhost.

- GRP_RADIUS_ACCT_PORT: radius accounting backend port. Default: 1813.

- GRP_RADIUS_TIMEOUT: timeout for radius responses. Default: 10s.

- GRP_IDLE_TIMEOUT: idle timeout for the TLS connection. Default: 60s.

## Install

```bash
$ make install
```

## Usage

```
Usage of grp:
  -v    Show version
```

## Note

Only tested with Golang 1.7 on Linux.
