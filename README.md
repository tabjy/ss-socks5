# ss-socks5
A shadowsocks server dialing through another SOCKS5 proxy, with shadowsocks-manager API support.

## Getting Started
ss-socks5 works like other shadowsocks server.

### Installing

Compile the latest version from source code:
```bash
$ go get -u -v github.com/tabjy/yagl/ss-socks5
```

Alternatively, you can download pre-built binaries from [release page](https://github.com/tabjy/ss-socks5/releases).

### Usage

Basic usage:
```bash
$ ss-socks5 -password <password> -socks5-address <socks5-server>
```
`socks5-address` is default to `localhost:1080` if not specified.

Using shadowsocks-manager API (multi-user):
```bash
$ ss-socks5 -manager-address <manager-address> -socks5-address <socks5-server>
```

For more options, inquiry help page with:
```bash
$ ss-socks5 -h
```
```
Usage of ss-socks5:
  -cipher string
    	encrypt method (default "AES-256-CFB")
  -log-level string
    	logging level (default "info")
  -manager-address string
    	address listening shadowsocks manager commands
  -password string
    	shadowsocks password for single user mode
  -server-address string
    	address of your server, (default "0.0.0.0:8388")
  -server-host string
    	hostname of your server (default "0.0.0.0")
  -socks5-address string
    	address of SOCKS5 serverAddr (default "localhost:1080")
```

## License

This project is licensed under the [MIT LICENSE](LICENSE), with exception that [tcp.go](3rd-party/go-shadowsocks2/tcp.go) is licensed under Apache 2.0.
