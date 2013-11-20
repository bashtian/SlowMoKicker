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

	serial "github.com/tarm/goserial"
)

const (
	maxPoints = 6
	goalTeam1 = 1
	goalTeam2 = 2
	tempFile  = "temp.mkv"
	outFile   = "output.mkv"
)

var debug = flag.Bool("d", false, "simulate input")
var videoInput = flag.String("v", "/dev/video0", "video input device")

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
				fmt.Fprintln(w, "1\r")

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
				log.Print(got.b)
				goal(got.b)
			}
		case <-timeout.C:
			//stop waiting for the reader to send something on channel rc
		}

		//time.Sleep(20 * time.Millisecond) //stutter the infinite loop.
	}
}

var lastGoal = time.Now()

var team1 = 0
var team2 = 0

var mplayer *exec.Cmd
var ffmpeg *exec.Cmd

func goal(team string) {
	if time.Since(lastGoal) > time.Second*3 {
		switch team {
		case "1\n":
			team1++
			fmt.Printf("team 1 score:%v\n", team1)
		case "2\n":
			team2++
			fmt.Printf("team 2 score:%v\n", team2)
		default:
			if team != "" {
				fmt.Printf("unknown output: %q\n", team)
			}

			return
		}

		lastGoal = time.Now()

		writeSrt()
		interruptRecording()
		if team1 >= maxPoints || team2 >= maxPoints {
			team1, team2 = 0, 0
		}
	}
}

func writeSrt() error {
	// s := fmt.Sprintf("1\n00:00:00,000 --> 99:00:00,000\n%v:%v\n", team1, team2)
	// return ioutil.WriteFile("score.srt", []byte(s), 0644)

	f, err := os.OpenFile("score.srt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	var t time.Time
	for i := 1; i <= 600; i++ {
		t2 := t.Add(time.Second * 1)
		fmt.Fprintf(f, "%v\n%v --> %v\n%v:%v\n\n", i, t.Format("15:04:05,000"), t2.Format("15:04:05,000"), team1, team2)
		t = t2
	}

	f.Close()
	return nil
}

func interruptRecording() {
	if ffmpeg != nil {
		err := ffmpeg.Process.Signal(os.Interrupt)
		if err != nil {
			log.Print("error killing ffmpeg:" + err.Error())
		}
	}
	//ffmpeg = startRecording()
}

func startRecordingLoop() {
	os.Remove(tempFile)
	for {
		log.Println("start recording")
		//ffmpeg = exec.Command("ffmpeg", "-f", "video4linux2", "-s", "640x480", "-r", "60", "-i", "/dev/video0", "-vcodec", "libx264", tempFile)
		ffmpeg = exec.Command("ffmpeg", "-f", "video4linux2", "-s", "640x480", "-r", "60", "-i", *videoInput, tempFile)

		b, err := ffmpeg.CombinedOutput()
		if err != nil && err.Error() != "exit status 255" {
			log.Fatal(b)
			//fmt.Printf("%s", b)
		}

		os.Remove(outFile)
		err = os.Rename(tempFile, outFile)
		if err != nil {
			log.Print("error renaming file:" + err.Error())
			continue
		}

		if mplayer != nil {
			mplayer.Process.Kill()
		}

		mplayer = startMplayer()
	}
}

/*func startRecording() *exec.Cmd {
	fmt.Println("start recording")
	//cmd := exec.Command("ffmpeg", "-f", "video4linux2", "-i", "/dev/video0", "-vcodec", "libx264", tempFile)
	cmd := exec.Command("ffmpeg", "-f", "video4linux2", "-i", "/dev/video1", tempFile)
	err := cmd.Start()
	if err != nil {
		log.Println("recording error? ", err)
	}
	return cmd
}*/

func startMplayer() *exec.Cmd {
	out, err := exec.Command("./length.sh").Output()
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%q", out)
	o := strings.Split(string(out), " ")
	fmt.Printf("The lenght is %s\n", out)
	l, err := strconv.ParseFloat(o[0], 64)
	if err != nil {
		log.Fatal(err)
	}

	newStart := strconv.FormatFloat(l-2.50, 'f', 6, 64)
	fmt.Println("start at", newStart)

	//cmd := exec.Command("mplayer", "-fs" ,"-fixed-vo", "-ss", newStart, "-endpos", "1.00", "-speed", "1/4", "-loop", "0", outFile)
	cmd := exec.Command("mplayer", "-fs", "-fixed-vo", "-ss", newStart, "-speed", "1/4", "-sub", "score.srt", "-loop", "0", outFile)
	//cmd := exec.Command("cvlc", "-L", "--rate", "0.25", "--start-time", newStart, "--sub-file=", "score.srt", "--input-fast-seek", outFile)
	cmd.Stdout = os.Stdout
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

func killChan(cmd exec.Cmd, kill chan bool) chan error {
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	go func() {
		<-kill
		if err := cmd.Process.Kill(); err != nil {
			log.Fatal("failed to kill: ", err)
		}
		log.Println("process killed")
	}()
	return done
}
