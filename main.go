package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"math/big"
	stdnet "net"
	"os"
	"time"
	"websocket_server/logs"
	"websocket_server/net"

	"go.uber.org/zap"
)

var (
	port  = flag.Int("port", 8888, "端口号")
	wss   = flag.Int("wss", 0, "是否开启wss，开启请填1，默认为0")
	ip    = flag.String("ip", "0.0.0.0", "ip地址")
	model = flag.String("model", "debug", "日志模式：debug/product")
)

func init() {
	flag.Parse()
	logs.InitLogger("./logs.log", zap.DebugLevel, *model == "debug")
	logs.InfoF("启动参数, port = %d, ip = %s\n", *port, *ip)
}

func generateCert(certPath, keyPath string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"AnimeBattle"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []stdnet.IP{stdnet.ParseIP("127.0.0.1"), stdnet.ParseIP("0.0.0.0")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	return nil
}

func main() {
	s := net.Server{}
	// 初始化
	s.InitServer()
	// 注册V3的接口实现
	// s.Register(extends.V3Api{})
	if *wss == 1 {
		if _, err := os.Stat("tls.pem"); os.IsNotExist(err) {
			if _, err := os.Stat("tls.key"); os.IsNotExist(err) {
				logs.InfoF("证书不存在，自动生成自签名证书...\n")
				err := generateCert("tls.pem", "tls.key")
				if err != nil {
					panic(err)
				}
				logs.InfoF("证书生成成功\n")
			}
		}
		s.ListenTLS(*ip, *port)
	} else {
		s.Listen(*ip, *port)
	}
}
