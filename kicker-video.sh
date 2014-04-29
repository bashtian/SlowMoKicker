#!/bin/bash

TMP_DIR=/tmp
OFFSET="0.5"
DURATION_LONG=3
DURATION_SHORT=1
SLOW_FACTOR=4

mkdir -p $TMP_DIR

function capture {
	rm -f $2
	/opt/ffmpeg/bin/ffmpeg -f video4linux2 -s 640x480 -r 60 -i $1 -c:v mpeg2video -q 2 -t 300 -y $2
}

function generate_mp4 {
	#pkill ffmpeg
	DURATION=`ffprobe -loglevel error -show_format -show_streams $1 | grep 'duration=' | tail -n 1 | awk -F \= {'print $2'}`

	CUT_LONG=`echo "$DURATION - $DURATION_LONG - $OFFSET" | bc`
	CUT_SHORT=`echo "($DURATION_LONG - $DURATION_SHORT) * $SLOW_FACTOR" | bc`
	DURATION_SHORT=`echo "$DURATION_SHORT * $SLOW_FACTOR" | bc`

	/opt/ffmpeg/bin/ffmpeg -r 60 -i $1 -ss $CUT_LONG -t $DURATION_LONG -c:v copy -y $TMP_DIR/cut.avi
	/opt/ffmpeg/bin/ffmpeg -i $TMP_DIR/cut.avi -c:v libx264 -preset ultrafast -f mp4 -qp 0 -y $2
	/opt/ffmpeg/bin/ffmpeg -i $TMP_DIR/cut.avi -ss $CUT_SHORT -t $DURATION_SHORT -vf "setpts=($SLOW_FACTOR)*PTS" -c:v libx264 -preset ultrafast -f mp4 -qp 0 -y $3
}

case $1 in
	capture)
		capture $2 $3
		;;
	generate-mp4)
		generate_mp4 $2 $3 $4
		;;
esac


#mplayer -fs -zoom -vo x11 $TMP_DIR/long.mp4
#mplayer -fs -zoom -vo x11 $TMP_DIR/slow.mp4
