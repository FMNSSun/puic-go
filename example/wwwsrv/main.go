package main

import (
	"flag"
	"net/http"
	"math/rand"
	"os"

	_ "net/http/pprof"

	"github.com/lucas-clemente/quic-go/h2quic"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/internal/utils"

	"github.com/mami-project/plus-lib"
)

func initHttp(prefix string) {
	http.HandleFunc(prefix + "256", func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 256)
		rand.Read(data)
		w.Write(data)
	})

	
	http.HandleFunc(prefix + "4KiB", func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 4096)
		rand.Read(data)
		w.Write(data)
	})


	http.HandleFunc(prefix + "1MiB", func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 1024 * 1024)
		rand.Read(data)
		w.Write(data)
	})

	
	http.HandleFunc(prefix + "16MiB", func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 16 * 1024 * 1024)
		rand.Read(data)
		w.Write(data)
	})

	http.HandleFunc(prefix + "128MiB", func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 128 * 1024 * 1024)
		rand.Read(data)
		w.Write(data)
	})
}

func main() {
	verbose := flag.Bool("v", false, "verbose")
	verbosePlus := flag.Bool("vp", false, "verbose plus?")
	laddr := flag.String("laddr", "localhost:6121", "Local address to listen on.")
	certFilePath := flag.String("cert", "cert.pem", "Path to certificate file (PEM)")
	keyFilePath := flag.String("key", "key.pem", "Path to key file (PEM) (unencrypted)")
	prefix := flag.String("prefix","/data/","Path prefix where the API methods should be available under (should start and end with a slash)")
	usePlus := flag.Bool("use-plus", true, "Use PLUS?")

	flag.Parse()

	quic.UsePLUS = *usePlus

	initHttp(*prefix)

	if *verbosePlus {
		PLUS.LoggerDestination = os.Stdout
	}

	if *verbose {
		utils.DefaultLogger.SetLogLevel(utils.LogLevelDebug)
	} else {
		utils.DefaultLogger.SetLogLevel(utils.LogLevelInfo)
	}

	certFile := *certFilePath
	keyFile := *keyFilePath

	err := h2quic.ListenAndServeQUIC(*laddr, certFile, keyFile, nil)

	if err != nil {
		panic(err)
	}
}
