CFLAGS=-ggdb -g3 -std=gnu2x -Wall -Wextra -pedantic -fsanitize=address
LDFLAGS=$(shell pkg-config --cflags --libs fuse3)

all:
	cc -o errorfs errorfs.c $(CFLAGS) $(LDFLAGS)

clean:
	rm -f errorfs

