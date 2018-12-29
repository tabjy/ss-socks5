package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tabjy/ss-socks5/3rd-party/go-shadowsocks2"

	"github.com/tabjy/ss-socks5/internal"
	"github.com/tabjy/yagl"
)

var Log yagl.Logger

var (
	serverAddr string
	password   string

	serverHost string
	mgrAddr    string

	cipher string

	s5Addr string

	logLevel string
)

func init() {
	flag.StringVar(&serverAddr, "server-address", "0.0.0.0:8388", "address of your server,")
	flag.StringVar(&password, "password", "", "shadowsocks password for single user mode")

	flag.StringVar(&serverHost, "server-host", "0.0.0.0", "hostname of your server")
	flag.StringVar(&mgrAddr, "manager-address", "", "address listening shadowsocks manager commands")

	flag.StringVar(&cipher, "cipher", "AES-256-CFB", "encrypt method")

	flag.StringVar(&s5Addr, "socks5-address", "localhost:1080", "address of SOCKS5 serverAddr")

	flag.StringVar(&logLevel, "log-level", "info", "logging level")
}

func main() {
	flag.Parse()
	initLogger()

	if mgrAddr == "" {
		// single user mode
		if password == "" {
			Log.Fatal("password is needed to run in single user mode")
		}

		go func() {
			internal.ServeAccount(serverAddr, s5Addr, cipher, &internal.Account{
				Port:     0,
				Password: password,
				Traffic:  0,
				Sig:      make(chan int),
			})

			os.Exit(1)
		}()
	} else {
		// multi user mode
		if serverHost == "" {
			Log.Fatal("server hostname is needed to run in multi user mode")
		}
		go func() {
			internal.MgrServer(mgrAddr, s5Addr, serverHost, cipher)

			os.Exit(1)
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func initLogger() {
	var lvl int

	switch logLevel {
	case "trace":
		lvl = yagl.LvlTrace
	case "debug":
		lvl = yagl.LvlDebug
	case "info":
		lvl = yagl.LvlInfo
	case "warn":
		lvl = yagl.LvlWarn
	case "error":
		lvl = yagl.LvlError
	case "panic":
		lvl = yagl.LvlPanic
	case "fatal":
		lvl = yagl.LvlFatal
	default:
		fmt.Fprintf(os.Stderr, "unrecognized logging lvl: %s\n", logLevel)
		os.Exit(1)
	}

	Log = yagl.New(
		yagl.FlgDate|yagl.FlgTime|yagl.FlgShortFile,
		lvl,
		os.Stderr,
	)

	internal.Log = Log
	go_shadowsocks2.Log = Log
}
