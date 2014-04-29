package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	serial "github.com/tarm/goserial"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	go h.run()
	//go startMplayer()
	if *debug {
		r, w := io.Pipe()

		//w bufio.NewWriter(wr)
		go func() {
			for {
				time.Sleep(15 * time.Second)
				fmt.Fprintln(w, "1\r")

			}
		}()
		go startMatch(r)

	} else {
		tty := flag.Arg(0)
		if tty == "" {
			tty = "/dev/ttyACM0"
		}
		log.Print("Opening serial port", tty)
		c := &serial.Config{Name: tty, Baud: 9600}
		s, err := serial.OpenPort(c)
		if err != nil {
			log.Fatal(err)
		}
		go startMatch(s)
	}

	http.Handle("/", http.FileServer(http.Dir("static/")))
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/restart", restartHandler)
	http.ListenAndServe(":8080", nil)
}
