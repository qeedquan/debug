all:
	cc -ggdb -g3 -ffunction-sections -c test.c
	cc -Wl,-T linker.ld -o test test.o

clean:
	rm -f test *.o
