package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	serial "github.com/tarm/goserial"
)

var webOnly = flag.Bool("w", true, "start only webserver")
var debug = flag.Bool("d", false, "simulate input")
var timeToRestart = flag.Int("t", 7, "time to restart after the game ended")
var videoInput = flag.String("v", "/dev/video0", "video input device")
var addr = flag.String("addr", ":8080", "http service address")

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	go h.run()
	//go startMplayer()
	if *webOnly {
		go startRecordingLoop()
	} else if *debug {
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
	http.Handle("/video/", http.StripPrefix("/video/", http.FileServer(http.Dir("video/"))))
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/restart", restartHandler)
	http.HandleFunc("/goal1", goal1Handler)
	http.HandleFunc("/goal2", goal2Handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(stats)
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	stats.ResetLastGoal()
	h.broadcast <- stats.TextBytes()
	//json.NewEncoder(w).Encode(stats)
}

func restartHandler(w http.ResponseWriter, r *http.Request) {
	stats.Restart()
	h.broadcast <- stats.TextBytes()
	//json.NewEncoder(w).Encode(stats)
}

func goal1Handler(w http.ResponseWriter, r *http.Request) {
	go goal("1")
	//json.NewEncoder(w).Encode(stats)
}

func goal2Handler(w http.ResponseWriter, r *http.Request) {
	go goal("2")
	//json.NewEncoder(w).Encode(stats)
}
