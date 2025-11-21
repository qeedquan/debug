TEXTADDR=0x60000000

all:
	cc -o code code.c -ggdb -g3 -static -Wl,--section-start=.text=$(TEXTADDR)
	cc -o jit jit.c -ggdb -g3 -static

clean:
	rm -f code jit
