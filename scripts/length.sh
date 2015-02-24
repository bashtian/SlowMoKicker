#!/bin/bash
ffprobe -v quiet -show_format output.mkv | grep duration | awk -F \= '{print $2}'