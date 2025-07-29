#define _GNU_SOURCE
#include <stdio.h>
#include <string.h>
#include <signal.h>
#include <sys/mman.h>
#include <dlfcn.h>
#include <unistd.h>
#include <errno.h>

static void *(*real_mmap)(void *addr, size_t length, int prot, int flags, int fd, off_t offset);

#define xprintf(...)                                             \
	do {                                                     \
		char str[8192];                                  \
		int n = snprintf(str, sizeof(str), __VA_ARGS__); \
		write(1, str, n);                                \
	} while (0)

static void
tracer(int sig, siginfo_t *si, void *u)
{
	xprintf("addr %#lx\n", (long)si->si_addr);
	if (mprotect(si->si_addr, 0x1000, PROT_READ | PROT_WRITE | PROT_EXEC) < 0) {
		xprintf("%s\n", strerror(errno));
		_exit(0);
	}
}

__attribute__((constructor)) static void
init(void)
{
	struct sigaction sa;

	printf("initializing mmap tracer\n");
	real_mmap = dlsym(RTLD_NEXT, "mmap");

	sa.sa_flags = SA_SIGINFO;
	sigemptyset(&sa.sa_mask);
	sa.sa_sigaction = tracer;
	sigaction(SIGSEGV, &sa, NULL);
}

void *
mmap(void *addr, size_t length, int prot, int flags, int fd, off_t offset)
{
	void *ptr;

	printf("mmap(addr = %p, length = %#x, prot = %x, flags = %x, fd = %d, off = %ld\n",
	       addr, length, prot, flags, fd, offset);
	ptr = real_mmap(addr, length, prot, flags, fd, offset);
	if (ptr == MAP_FAILED)
		return ptr;
	mprotect(ptr, length, PROT_WRITE);
	return ptr;
}
