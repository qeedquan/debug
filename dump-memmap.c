#define _GNU_SOURCE
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <stdarg.h>
#include <inttypes.h>
#include <errno.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <sys/ptrace.h>
#include <fcntl.h>
#include <unistd.h>
#include <limits.h>
#include <getopt.h>

void
fatal(const char *fmt, ...)
{
	va_list ap;

	fprintf(stderr, "dump-memmap: ");
	va_start(ap, fmt);
	vfprintf(stderr, fmt, ap);
	va_end(ap);
	fprintf(stderr, "\n");
	exit(1);
}

void
usage(void)
{
	fprintf(stderr, "usage: dump-memmap pid file\n");
	exit(2);
}

int
main(int argc, char *argv[])
{
	char path[PATH_MAX], line[1024];
	FILE *mfd;
	void *buf;
	int rfd, wfd, pid, c, pgsz;

	while ((c = getopt(argc, argv, "h")) != -1) {
		switch (c) {
		case 'h':
		default:
			usage();
			break;
		}
	}

	argc -= optind;
	argv += optind;
	if (argc < 2)
		usage();

	pid = atoi(argv[0]);
	if (ptrace(PTRACE_ATTACH, pid, 0, 0) < 0)
		fatal("failed to attach to process %d: %s\n", pid, strerror(errno));

	snprintf(path, sizeof(path), "/proc/%d/maps", pid);
	mfd = fopen(path, "rb");
	if (mfd == NULL)
		fatal("%s: %s", path, strerror(errno));

	snprintf(path, sizeof(path), "/proc/%d/mem", pid);
	rfd = open(path, O_RDONLY);
	if (rfd < 0)
		fatal("%s: %s", path, strerror(errno));

	wfd = open(argv[1], O_WRONLY | O_CREAT, 0644);
	if (wfd < 0)
		fatal("%s: %s", argv[1], strerror(errno));

	pgsz = getpagesize();
	buf = malloc(pgsz);
	if (buf == NULL)
		fatal("%s", strerror(errno));

	while (fgets(line, sizeof(line), mfd)) {
		char perm[32];
		unsigned long long addr, start, end, size, off;
		int n;

		pgsz = getpagesize();
		n = sscanf(line, "%llx-%llx %31s %llx", &start, &end, perm, &off);
		if (n != 4)
			continue;

		size = end - start;
		printf("%llx-%llx %llx %s %llx\n", start, end, size, perm, off);
		if (strchr(perm, 'r') == NULL)
			continue;

		lseek(rfd, start, SEEK_SET);
		for (addr = start; addr < end; addr += pgsz) {
			n = read(rfd, buf, pgsz);
			if (n < 0)
				printf("%s\n", strerror(errno));
			write(wfd, buf, pgsz);
		}
	}

	fclose(mfd);
	close(rfd);
	close(wfd);

	ptrace(PTRACE_CONT, pid, 0, 0);
	ptrace(PTRACE_DETACH, pid, 0, 0);

	return 0;
}
