#!/bin/sh

ARCH=i386
FORMAT=elf64-x86-64

usage() {
	echo "Usage: input output"
	exit 2
}

if [ $# -ne 2 ]; then
	usage
fi

objcopy -I binary -O $FORMAT --binary-architecture $ARCH $1 $2
