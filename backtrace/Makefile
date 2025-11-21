CFLAGS=-Wall -Wextra -pedantic -ggdb -g3
LDFLAGS=-ldw

all:
	cc -o backtrace-dwfl backtrace-dwfl.c $(CFLAGS) $(LDFLAGS)
	cc -o backtrace-glibc backtrace-glibc.c $(CFLAGS) $(LDFLAGS)

clean:
	rm -f backtrace-dwfl backtrace-glibc
