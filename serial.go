package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/goserial"
)

const (
	maxPoints = 6
	goalTeam1 = 1
	goalTeam2 = 2
	tempFile  = "temp.avi"
	outFile   = "output.avi"
)

var debug = flag.Bool("d", false, "simulate input")

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	//go startMplayer()
	if *debug {
		r, w := io.Pipe()

		//w bufio.NewWriter(wr)
		go func() {
			for {
				time.Sleep(15 * time.Second)
				fmt.Fprintln(w, "1")

			}
		}()
		startMatch(r)

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
		startMatch(s)
	}

}

func startMatch(reader io.Reader) {
	//ffmpeg = startRecording()
	go startRecordingLoop()
	type readResult struct {
		b   string
		err error
	}
	rc := make(chan *readResult)
	r := bufio.NewReader(reader)
	go func() {
		time.Sleep(time.Second * 3)
		for {

			//log.Print("test")
			s, err := r.ReadString('\n')
			//log.Print("Reading from serial port ", s)
			//if s == "1" || s == "2" {
			rc <- &readResult{s, nil}
			//}

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
				//log.Print(got.b)
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

var mplayer *exec.Cmd
var ffmpeg *exec.Cmd

func goal(team string) {
	if time.Since(lastGoal) > time.Second*2 {
		switch team {
		case "1\n":
			team1++
			fmt.Printf("team 1 score:%v\n", team1)
		case "2\n":
			team2++
			fmt.Printf("team 2 score:%v\n", team2)
		default:
			fmt.Printf("unknown output: %q\n", team)
			return
		}

		lastGoal = time.Now()

		writeSrtAndKill()
		handleRecordAndPlaying()
		if team1 >= maxPoints || team2 >= maxPoints {
			team1, team2 = 0, 0
		}
	}
}

func handleRecordAndPlaying() {
	if ffmpeg != nil {
		err := ffmpeg.Process.Kill()
		if err != nil {
			log.Print("error killing ffmpeg:" + err.Error())
		}
	}
	os.Remove(outFile)
	err := os.Rename(tempFile, outFile)
	if err != nil {
		log.Print("error renaming file:" + err.Error())
	}

	if mplayer != nil {
		mplayer.Process.Kill()
	}

	mplayer = startMplayer()
	//ffmpeg = startRecording()
}

func writeSrtAndKill() error {
	f, err := os.OpenFile("score.srt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	fmt.Fprintf(f, "1\n00:00:00,000 --> 99:00:00,000\n%v:%v\n", team1, team2)
	//n, err := f.Write(data)
	f.Close()

	//exec.Command("killall", "ffmpeg").Start()
	return nil
}

func startRecordingLoop() {
	os.Remove(tempFile)
	for {
		fmt.Println("start recording")
		ffmpeg = exec.Command("ffmpeg", "-f", "video4linux2", "-s", "640x480", "-r", "60", "-an", "-i", "/dev/video0", tempFile)
		b, err := ffmpeg.CombinedOutput()
		if err != nil {
			//log.Fatal(err)
		}
		fmt.Printf("%s", b)
	}
}

func startRecording() *exec.Cmd {
	fmt.Println("start recording")
	cmd := exec.Command("ffmpeg", "-f", "video4linux2", "-s", "640x480", "-r", "60", "-an", "-i", "/dev/video0", tempFile)
	err := cmd.Start()
	if err != nil {
		log.Println("recording error? ", err)
	}
	return cmd
}

func startMplayer() *exec.Cmd {
	out, err := exec.Command("./length.sh").Output()
	if err != nil {
		log.Fatal(err)
	}
	o := strings.Split(string(out), " ")
	fmt.Printf("The lenght is %s\n", out)
	l, err := strconv.ParseFloat(o[0], 64)
	if err != nil {
		log.Fatal(err)
	}

	newStart := strconv.FormatFloat(l-2.00, 'f', 6, 64)
	fmt.Println("start at", newStart)

	//cmd := exec.Command("mplayer", "-fs" ,"-fixed-vo", "-ss", newStart, "-endpos", "1.00", "-speed", "1/4", "-loop", "0", outFile)
	cmd := exec.Command("mplayer", "-fixed-vo", "-speed", "4/1", "-loop", "0", outFile)

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	return cmd
}

func killAfterTime(cmd exec.Cmd) {
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(3 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			log.Fatal("failed to kill: ", err)
		}
		<-done // allow goroutine to exit
		log.Println("process killed")
	case err := <-done:
		log.Printf("process done with error = %v", err)
	}

}
