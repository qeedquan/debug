#define _GNU_SOURCE
#include <stdio.h>
#include <errno.h>
#include <unistd.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <err.h>

int
main(void)
{
	int fd;
	unsigned char *p;

	fd = open("/dev/gpiomem", O_RDONLY);
	if (fd < 0)
		err(1, "open");

	p = mmap(NULL, getpagesize(), PROT_READ, MAP_SHARED, fd, 0x10000);
	if (p == MAP_FAILED)
		err(1, "mmap");
	p[0] = 1;

	return 0;
}
