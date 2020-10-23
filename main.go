package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
)
import "github.com/prometheus/client_golang/prometheus/promhttp"

const VERSION  = 0.2

func helpText() {
	fmt.Print("# https://github.com/vvampirius/bibExchangesBot\n\n")
	flag.PrintDefaults()
}

func Pong(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `PONG`)
}

func main() {
	cwd, _ := os.Getwd()
	help := flag.Bool("h", false, "print this help")
	ver := flag.Bool("v", false, "Show version")
	listen := flag.String("l", `:8080`, "Listen on [address]<:port>")
	token := flag.String("t", os.Getenv(`TOKEN`), "Telegram token")
	storagePath := flag.String("s", cwd, "Path to store files")
	webHook := flag.String("w", os.Getenv(`WEBHOOK`), "Callback URL (webHook)")
	testParsing := flag.Bool("p", false, "Just try to get rates from belinvestbank.by")
	flag.Parse()

	if *help {
		helpText()
		os.Exit(0)
	}

	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	log.SetFlags(log.Lshortfile)

	if *testParsing {
		GetExchangeTest()
		os.Exit(0)
	}

	if *token == `` {
		fmt.Fprintln(os.Stderr, `You must define TOKEN!`)
		syscall.Exit(1)
	}

	if *webHook == `` {
		fmt.Fprintln(os.Stderr, `You must define callback URL (webHook)!`)
		syscall.Exit(1)
	}

	log.Printf("Starting version %g...\n", VERSION)

	core, err := NewCore(*storagePath, *token, *webHook)
	if err != nil { os.Exit(1) }

	server := http.Server{ Addr: *listen }
	http.HandleFunc(`/ping`, Pong)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc(`/`, core.httpHandler)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err.Error())
	}
}
