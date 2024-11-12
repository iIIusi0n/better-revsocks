## Better Revsocks
Reverse SOCKS5 server that multiplex connections using Yamux.
Rust for client, Golang for server.

### Usage
1. Start server daemon
   
   `revsocks start [-p <port>] [--tls] [--tor]` or `revsocks run [-p <port>] [--tls] [--tor]`

2. Connect client to server

   `client <host> <port>`

3. List up connected clients

   `revsocks ls`

4. Close client from server

   `revsocks close <id>`

### TODO
- [x] Multiplexing using Yamux
- [x] Agent connection health check
- [x] TLS support
- [ ] Tor support with Arti (experimental)
- [x] Rich CLI with daemon

