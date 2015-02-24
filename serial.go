package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	maxPoints = 6
	goalTeam1 = 1
	goalTeam2 = 2
	tempFile  = "temp.mkv"
	outFile   = "output.mkv"
)

//var lastGoalTime = time.Now()
//var lastGoalTeam = 0

var mplayer *exec.Cmd
var ffmpeg *exec.Cmd

var stopRecording = make(chan bool)

var stats = Stats{LastGoalTime: time.Now()}

func startMatch(reader io.Reader) {
	//ffmpeg = startRecording()
	go startRecordingLoop()
	type readResult struct {
		b   string
		err error
	}
	rc := make(chan *readResult, 100)
	r := bufio.NewReader(reader)
	go func() {
		//stats.LastGoalTime = time.Now()
		//time.Sleep(time.Second * 3)
		for {
			//log.Print("test")
			bytes, _, err := r.ReadLine()
			//log.Print("Reading from serial port ", s)
			//if s == "1" || s == "2" {
			rc <- &readResult{string(bytes), nil}
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
				//log.Println("channel:", got.b)
				goal(got.b)
			}
		case <-timeout.C:
			//stop waiting for the reader to send something on channel rc
		}

		time.Sleep(200 * time.Millisecond) //stutter the infinite loop.
	}
}

func goal(team string) {
	//log.Println("goal:", team)
	if time.Since(stats.LastGoalTime) > time.Second*3 {
		log.Println("got goal")
		stats.LastGoalTime = time.Now()

		//go writeSrt()
		stopRecording <- true
		switch {
		case strings.Contains(team, "1"):
			stats.Team1++
			stats.LastGoalTeam = goalTeam1
			sendMessage()
			log.Printf("team 1 score:%v\n", stats.Team1)
		case strings.Contains(team, "2"):
			stats.Team2++
			stats.LastGoalTeam = goalTeam2
			sendMessage()
			log.Printf("team 2 score:%v\n", stats.Team2)
		default:
			if team != "" {
				fmt.Printf("unknown output: %q\n", team)
			}
			return
		}

		if stats.IsFinshed() {
			finishGame()
		}
	}
}

func finishGame() {
	go func() {
		h.broadcast <- stats.TextBytes()
		time.Sleep(time.Second * time.Duration(*timeToRestart))
		stats.Restart()
		h.broadcast <- stats.TextBytes()
	}()
}

func sendMessage() {
	h.broadcast <- stats.TextBytes()
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
		fmt.Fprintf(f, "%v\n%v --> %v\n%v:%v\n\n", i, t.Format("15:04:05,000"), t2.Format("15:04:05,000"), stats.Team1, stats.Team2)
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
	//os.Remove(tempFile)
	for {
		//ffmpeg = exec.Command("ffmpeg", "-f", "video4linux2", "-s", "640x480", "-r", "60", "-i", "/dev/video0", "-vcodec", "libx264", tempFile)
		errorChan := make(chan error)
		successChan := make(chan bool)
		filename := fmt.Sprintf("video-%v.avi", time.Now().Unix())
		go func() {
			log.Println("start recording")

			//ffmpeg = exec.Command("./kicker-video.sh", "capture", *videoInput, filename)
			//ffmpeg -f video4linux2 -s 640x480 -r 60 -i $1 -c:v mpeg2video -q 2 -t 300 -y $2
			ffmpeg = exec.Command("ffmpeg", "-f", "video4linux2", "-s", "640x480", "-i", *videoInput, "-r", "60", "-t", "300", "-y", filename)
			b, err := ffmpeg.CombinedOutput()
			if err != nil && err.Error() != "exit status 255" {
				log.Print(string(b))
				//fmt.Printf("%s", b)
				errorChan <- err
			} else {
				successChan <- true
			}

		}()

		select {
		case <-stopRecording:
			log.Println("stopRecording")
			interruptRecording()
			<-successChan
			go convertFile(filename)
		// a read from ch has occurred
		case err := <-errorChan:
			log.Println("got error chan")
			log.Fatal(err)
			// the read from ch has timed out
		case <-successChan:
			log.Println("ffmpeg timelimit reached")
			// ffmepg timelimit
		}

		// os.Remove(outFile)
		// err := os.Rename(tempFile, outFile)
		// if err != nil {
		// 	log.Print("error renaming file:" + err.Error())
		// 	continue
		// }

		// if mplayer != nil {
		// 	mplayer.Process.Kill()
		// }

		//mplayer = startMplayer()
	}
}

func convertFile(filename string) {
	log.Println("converting", filename)
	cmd := exec.Command("./kicker-video.sh", "generate-mp4", filename, "video/normal.mp4", "video/slow.mp4")
	err := cmd.Start()
	if err != nil {
		log.Println("generate error:", err)
	}
	log.Println("converting finished", filename)
	h.broadcast <- []byte("http://localhost:8081/slow.mp4")
	h.broadcast <- stats.TextBytes()
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
	fmt.Printf("The lenght is %s\n", out)
	o1 := strings.Split(string(out), "\n")
	o2 := strings.Split(o1[0], " ")
	l, err := strconv.ParseFloat(o2[0], 64)
	if err != nil {
		log.Fatal(err)
	}

	newStart := strconv.FormatFloat(l-2.50, 'f', 6, 64)
	fmt.Println("start at", newStart)

	//cmd := exec.Command("mplayer", "-fs" ,"-fixed-vo", "-ss", newStart, "-endpos", "1.00", "-speed", "1/4", "-loop", "0", outFile)
	cmd := exec.Command("mplayer", "-zoom", "-fs", "-fixed-vo", "-ss", newStart, "-speed", "1/4", "-sub", "score.srt", "-loop", "0", outFile)
	//cmd := exec.Command("cvlc", "-L", "--rate", "0.25", "--start-time", newStart, "--sub-file=", "score.srt", "--input-fast-seek", outFile)
	//cmd.Stdout = os.Stdout
	err = cmd.Start()
	if err != nil {
		//log.Fatal(err)
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
