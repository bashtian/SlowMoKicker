#!/bin/bash


nohup mplayer -fs -fixed-vo -loop 0 -sub score.srt -subfont-text-scale 7 output.avi &
#nohup mplayer -fixed-vo -loop 0 -sub score.srt -subfont-text-scale 7 output.avi &

while true
do

	cd /home/sebastian/Kicker

	rm out.avi ; ffmpeg -f video4linux2 -s 640x480 -r 60 -an -i /dev/video1 out.avi
	#rm out.avi ; ffmpeg -f video4linux2 -s 640x480 -an -i /dev/video0 out.avi

	CUR_DUR="`ffprobe -show_format out.avi |grep duration | awk -F \= '{print $2}'`"

	LAG="1"
	START_SLOW=$(echo "scale=2; $CUR_DUR - $LAG - 1;"|bc)
	END_SLOW=$(echo "scale=2; $CUR_DUR - $LAG;"|bc)

	echo ""
	echo $CUR_DUR
	echo $START_SLOW
	echo $END_SLOW


	#echo "mencoder -speed 1/4 out.avi -ovc copy -nosound -ss $START_SLOW -endpos $END_SLOW -o output.avi"
	mencoder -speed 1/4 out.avi -ovc copy -nosound -ss $START_SLOW -endpos $END_SLOW -o output.avi

done
