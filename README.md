# cardano-node-api

Cardano Node API

An HTTP API for interfacing with a local Cardano Node and providing the node
internal data for HTTP clients. This service communicates with a Cardano
full node using the Ouroboros network protocol via a UNIX socket and exposes
the underlying Node-to-Client (NtC) Ouroboros mini-protocols to clients via
a REST API or UTxO RPC gRPC API.

## Usage

The recommended method of using this application is via the published
container images, coupled with Blink Labs container images for the Cardano
Node.

```
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v <mount for cardano-node IPC> \
  ghcr.io/blinklabs-io/cardano-node-api:main
```

<!-- Binaries can be executed directly and are available from
[Releases](https://github.com/blinklabs-io/cardano-node-api/releases).

```
./cardano-node-api
```
-->
### Configuration

Configuration can be done using either a `config.yaml` file or setting
environment variables. Our recommendation is environment variables to adhere
to the 12-factor application philisophy.

#### Environment variables

Configuration via environment variables can be broken into two sets of
variables. The first set controls the behavior of the application, while the
second set controls the connection to the Cardano node instance.

Application configuration:
- `API_LISTEN_ADDRESS` - Address to bind for API calls, all addresses if empty
    (default: empty)
- `API_LISTEN_PORT` - Port to bind for API calls (default: 8080)
- `DEBUG_ADDRESS` - Address to bind for pprof debugging (default: localhost)
- `DEBUG_PORT` - Port to bind for pprof debugging, disabled if 0 (default: 0)
- `GRPC_LISTEN_ADDRESS` - Address to bind for UTxO RPC gRPC, all addresses if empty
    (default: empty)
- `GRPC_LISTEN_PORT` - Port to bind for gRPC calls (default: 9090)
- `LOGGING_HEALTHCHECKS` - Log requests to `/healthcheck` endpoint (default: false)
- `LOGGING_LEVEL` - Logging level for log output (default: info)
- `METRICS_LISTEN_ADDRESS` - Address to bind for Prometheus format metrics, all
    addresses if empty (default: empty)
- `METRICS_LISTEN_PORT` - Port to bind for metrics (default: 8081)
- `TLS_CERT_FILE_PATH` - SSL certificate to use, requires `TLS_KEY_FILE_PATH`
    (default: empty)
- `TLS_KEY_FILE_PATH` - SSL certificate key to use (default: empty)

Configuring the TLS certificate and key paths will enable TLS on both the REST
API and the gRPC interface.

Connection to the Cardano node can be performed using specific named network
shortcuts for known network magic configurations. Supported named networks are:

- mainnet
- preprod
- preview
- sanchonet

You can set the network to an empty value and provide your own network magic to
connect to unlisted networks.

TCP connection to a Cardano Node without using an intermediary like SOCAT is
possible using the node address and port. It is up to you to expose the node's
NtC communication socket over TCP. TCP connections are preferred over socket
within the application.

Cardano node configuration:
- `CARDANO_NETWORK` - Use a named Cardano network (default: mainnet)
- `CARDANO_NODE_NETWORK_MAGIC` - Cardano network magic (default: automatically
    determined from named network)
- `CARDANO_NODE_SOCKET_PATH` - Socket path to Cardano node NtC via UNIX socket
    (default: /node-ipc/node.socket)
- `CARDANO_NODE_SOCKET_TCP_HOST` - Address to Cardano node NtC via TCP
   (default: unset)
- `CARDANO_NODE_SOCKET_TCP_PORT` - Port to Cardano node NtC via TCP (default:
    unset)
- `CARDANO_NODE_SOCKET_TIMEOUT` - Sets a timeout in seconds for waiting on
   requests to the Cardano node (default: 30)

### Connecting to a cardano-node

You can connect to either a cardano-node running locally on the host or a
container running either `inputoutput/cardano-node` or
`ghcr.io/blinklabs-io/cardano-node` by mapping in the correct paths and setting
the environment variables or configuration options to match.

#### Together with ghcr.io/blinklabs-io/cardano-node in Docker

Use Docker to run both cardano-node and cardano-node-api with Docker
volumes for blockchain storage and node-ipc.

```
# Start mainnet node
docker run --detach \
  --name cardano-node \
  -v node-data:/opt/cardano/data \
  -v node-ipc:/opt/cardano/ipc \
  -p 3001:3001 \
  ghcr.io/blinklabs-io/cardano-node run

# Start cardano-node-api
docker run --detach \
  --name cardano-node-api \
  -v node-ipc:/node-ipc \
  -p 8080:8080 \
  -p 9090:9090 \
  ghcr.io/blinklabs-io/cardano-node-api:main
```

#### Using a local cardano-node

Use the local path when mapping the node-ipc volume into the container to use
a local cardano-node.

```
# Start cardano-node-api
docker run --detach \
  --name cardano-node-api \
  -v /opt/cardano/ipc:/node-ipc \
  -p 8080:8080 \
  -p 9090:9090 \
  ghcr.io/blinklabs-io/cardano-node-api:main
```

## Development

There is a Makefile to provide some simple helpers.

Run from checkout:
```
go run .
```

Create a binary:
```
make
```

Create a docker image:
```
make image
```
