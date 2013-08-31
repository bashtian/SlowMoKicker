package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/tarm/goserial"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const (
	maxPoints = 6
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
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

	type readResult struct {
		b   string
		err error
	}
	rc := make(chan *readResult)
	r := bufio.NewReader(s)
	go func() {
		time.Sleep(time.Second * 3)
		for {

			//log.Print("test")
			s, err := r.ReadString('\n')

			//log.Print("Reading from serial port ", s)
			if s == "1" || s == "2" {
				rc <- &readResult{s, nil}
			}

			if err != nil && err != io.EOF {
				log.Printf("error: %v", err)
				return
			}
		}
	}()
	for {
		timeout := time.NewTicker(10 * time.Second)
		defer timeout.Stop() //is this necessary?

		select {
		case got := <-rc:
			//log.Print("got a result")
			switch {
			case got.err != nil:
				//Catching an EOF error here can indicate the port was disconnected.
				// -- if using a USB to serial port, and the device is unplugged
				//    while being read, we'll receive an EOF.
				log.Fatal("  error:" + got.err.Error())
			default:
				log.Print(got.b)
				goal(got.b)
			}
		case <-timeout.C:
			//stop waiting for the reader to send something on channel rc
		}

		time.Sleep(100 * time.Millisecond) //stutter the infinite loop.
	}

}

var lastGoal = time.Now()

var team1 = 0
var team2 = 0

func goal(team string) {
	if time.Since(lastGoal) > time.Second*2 {
		lastGoal = time.Now()

		if team == "1" {
			team1++
			fmt.Printf("team 1 score:%v\n", team1)
		} else {
			team2++
			fmt.Printf("team 2 score:%v\n", team2)
		}

		writeSrtAndKill()
		if team1 >= maxPoints || team2 >= maxPoints {
			team1, team2 = 0, 0
		}
	}

}

func writeSrtAndKill() error {
	f, err := os.OpenFile("score.srt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	fmt.Fprintf(f, `1
00:00:00,000 --> 99:00:00,000
%v:%v
`, team1, team2)
	//n, err := f.Write(data)
	f.Close()

	exec.Command("killall", "ffmpeg").Start()
	return nil
}
