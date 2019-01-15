#!/bin/bash
if [ -n "$(gofmt -l .)" ]; then
	echo "Go code is not formatted:"
	echo "########################"
	gofmt -d .
	echo "########################"
	exit 1
fi
