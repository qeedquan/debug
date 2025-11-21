all:
	cc -ggdb -g3 -o test test.c
	objcopy --only-keep-debug test test.sym
	objcopy --strip-debug test test_strip

clean:
	rm -f test test_strip test.sym
