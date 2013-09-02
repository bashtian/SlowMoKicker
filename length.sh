#!/bin/bash
ffprobe -v quiet -show_format output.avi | grep duration | awk -F \= '{print $2}'