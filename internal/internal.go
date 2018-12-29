package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/tabjy/ss-socks5/3rd-party/go-shadowsocks2"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/tabjy/yagl"
)

type Account struct {
	Port     int
	Password string
	Traffic  int64
	Sig      chan int
}

func (a *Account) IncrementTraffic(n int64) {
	a.Traffic += n
}

func (a *Account) SigChan() chan int {
	return a.Sig
}

var Log yagl.Logger // set by main.go

func ServeAccount(ssAddr, s5Addr, cipher string, account *Account) {
	var key []byte
	ciph, err := core.PickCipher(cipher, key, account.Password)
	if err != nil {
		log.Fatal(err)
	}

	go_shadowsocks2.TcpRemote(ssAddr, ciph.StreamConn, s5Addr, account)
}

func MgrServer(mgrAddr, s5Addr, serverHost, cipher string) {
	pc, err := net.ListenPacket("udp", mgrAddr)
	if err != nil {
		Log.Fatalf("UDP listen error: %v", err)
	}
	defer pc.Close()

	accounts := make(map[int]*Account)

	buf := make([]byte, 65507)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			Log.Errorf("UDP read error: %v", err)
			continue
		}

		req := strings.Replace(string(buf[:n]), "\n", "", -1)
		stop := strings.Index(req, ":")
		cmd := ""
		if stop == -1 {
			cmd = req
		} else {
			cmd = req[:stop]
		}

		rawPayload := req[stop+1:]

		res := "err"
		switch cmd {
		case "add":
			payload := make(map[string]interface{})
			if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
				Log.Errorf("JSON parse error: %v", err)
				break
			}

			portRaw, _ := payload["server_port"]
			passRaw, _ := payload["password"]

			portF, portTypeOk := portRaw.(float64)
			pass, passTypeOk := passRaw.(string)
			if !portTypeOk || !passTypeOk {
				Log.Errorf("JSON type error")
				break
			}
			port := int(portF)

			if _, found := accounts[port]; found {
				Log.Errorf("account already exists")
				break
			}

			account := &Account{
				Port:     port,
				Password: pass,
				Traffic:  0,
				Sig:      make(chan int),
			}
			accounts[port] = account

			go ServeAccount(fmt.Sprintf("%s:%d", serverHost, port), s5Addr, cipher, account)

			res = "ok"
		case "remove":
			payload := make(map[string]interface{})
			if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
				Log.Fatalf("JSON parse error: %v", err)
			}

			portRaw, portOk := payload["server_port"]
			portF, portTypeOk := portRaw.(float64)
			if !portOk || !portTypeOk {
				Log.Errorf("JSON payload error")
				break
			}
			port := int(portF)

			if _, found := accounts[port]; !found {
				Log.Errorf("account not found")
				break
			}

			go func() {
				accounts[port].Sig <- 0
			}()

			res = "ok"
		case "ping":
			var sb strings.Builder
			sb.WriteString("stat: {")
			tmp := ""
			for _, v := range accounts {
				if tmp != "" {
					sb.WriteString(tmp)
					sb.WriteString(", ")
				}

				tmp = fmt.Sprintf(`"%d": %d`, v.Port, v.Traffic)
			}
			sb.WriteString(tmp)
			sb.WriteString("}")
			res = sb.String()
		case "list":
			var sb strings.Builder
			sb.WriteString("[\n")
			tmp := ""
			for _, v := range accounts {
				if tmp != "" {
					sb.WriteString(tmp)
					sb.WriteString(", \n")
				}

				tmp = fmt.Sprintf("\t{\"server_port\": \"%d\", \"password\": \"%s\", \"method\": \"%s\"}", v.Port, v.Password, cipher)
			}
			sb.WriteString(tmp)
			sb.WriteString("\n]")
			res = sb.String()
		default:
			Log.Errorf("unknown command: bytes=%d from=%s command=%s payload=%s\n", n, addr.String(), cmd, rawPayload)
			continue
		}

		if res == "err" {
			Log.Errorf("invalid command: bytes=%d from=%s command=%s payload=%s\n", n, addr.String(), cmd, rawPayload)
		} else {
			Log.Infof("`%s` command received: bytes=%d from=%s payload=%s\n", cmd, n, addr.String(), rawPayload)
		}

		n, err = pc.WriteTo([]byte(res), addr)
		if err != nil {
			Log.Fatalf("UDP write error: %v", err)
		}

		Log.Infof("response sent: bytes=%d to=%s content=%s", n, addr.String(), res)
	}
}
