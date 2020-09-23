#define _GNU_SOURCE
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <unistd.h>
#include <assert.h>
#include <errno.h>
#include <limits.h>
#include <time.h>
#include <pthread.h>
#include <dlfcn.h>

typedef uint8_t u8;
typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;

typedef unsigned int uint;

typedef struct Track Track;

struct Track {
	int fd;
	void *addr;
	size_t alignment;
	size_t size;
	Track *next;
};

#define xprintf(...)                                                                                     \
	do {                                                                                             \
		char str[PATH_MAX], buf[8192];                                                           \
		snprintf(str, sizeof(str), __VA_ARGS__);                                                 \
		int n = snprintf(buf, sizeof(buf), "[%ld] [%ld] %s\n", pthread_self(), time(NULL), str); \
		write(1, buf, n);                                                                        \
		if (lfd > 0)                                                                             \
			write(lfd, buf, n);                                                              \
	} while (0)

#define min(a, b) (((a) < (b)) ? (a) : (b))

static int mkdirp(const char *);
static char *rpath(const char *);

static void *(*real_malloc)(size_t);
static void *(*real_realloc)(void *, size_t);
static void *(*real_calloc)(size_t, size_t);
static void *(*real_memalign)(size_t, size_t);
static void (*real_free)(void *);

static pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;
static int phase;
static Track *tracks;
static char *rootdir;
static int lfd;
static uint wid;

char __enable_dump_malloc = 1;

static void
init(void)
{
	if (phase >= 1)
		return;

	phase++;
	if (phase == 1) {
		rootdir = getenv("MALLOC_DIR");
		if (rootdir == NULL)
			rootdir = "dump_malloc";
		if (mkdirp(rootdir) < 0)
			xprintf("failed to make directory %s: %s", rootdir, strerror_l(errno, 0));
		lfd = creat(rpath("log.txt"), 0644);

		real_malloc = dlsym(RTLD_NEXT, "malloc");
		real_realloc = dlsym(RTLD_NEXT, "realloc");
		real_calloc = dlsym(RTLD_NEXT, "calloc");
		real_memalign = dlsym(RTLD_NEXT, "memalign");
		real_free = dlsym(RTLD_NEXT, "free");
	}
}

static void *
xmalloc(size_t size)
{
	void *ptr;
	if (real_malloc == NULL)
		ptr = sbrk(size);
	else
		ptr = real_malloc(size);
	memset(ptr, 0, size);
	return ptr;
}

static void
freetrack(Track *t)
{
	if (t == NULL)
		return;
	if (t->fd > 0)
		close(t->fd);
	if (t->addr != NULL && t->addr != MAP_FAILED)
		munmap(t->addr, t->size);
}

static void *
addtrack(const char *func, void *ret, void *frame, size_t alignment, size_t size)
{
	pthread_mutex_lock(&mutex);
	Track *t = xmalloc(sizeof(Track));
	if (t == NULL) {
		xprintf("failed to allocate track: %s", strerror_l(errno, 0));
		goto error;
	}
	t->alignment = alignment;
	t->size = size;

	char name[80];
	snprintf(name, sizeof(name), "a%u-s%zu-l%zu", wid, size, alignment);
	t->fd = open(rpath(name), O_RDWR | O_CREAT, 0644);
	if (t->fd < 0) {
		xprintf("failed to open file %s for mmap: %s", rpath(name), strerror_l(errno, 0));
		goto error;
	}

	if (size == 0)
		size = 1;
	ftruncate(t->fd, size);

	t->addr = mmap(NULL, size, PROT_READ | PROT_EXEC | PROT_WRITE, MAP_SHARED, t->fd, 0);
	if (t->addr == MAP_FAILED) {
		xprintf("failed to mmap %s: %s", name, strerror_l(errno, 0));
		goto error;
	}
	memset(t->addr, 0, size);

	xprintf("%s(id=%u frame=%p ret=%p addr=%p size=%zu alignment=%zu)",
	        func, wid, frame, ret, t->addr, size, alignment);
	wid++;

	if (tracks == NULL)
		tracks = t;
	else {
		t->next = tracks;
		tracks = t;
	}

	pthread_mutex_unlock(&mutex);
	return t->addr;

error:
	freetrack(t);
	pthread_mutex_unlock(&mutex);
	return NULL;
}

static void *
lookuptrack(void *addr)
{
	Track *p = NULL;
	pthread_mutex_lock(&mutex);
	for (Track *t = tracks; t; t = t->next) {
		if (t->addr == addr) {
			p = t;
			break;
		}
	}
	pthread_mutex_unlock(&mutex);
	return p;
}

void *
memalign(size_t alignment, size_t size)
{
	init();

	if (!__enable_dump_malloc && real_memalign)
		return real_memalign(alignment, size);

	return addtrack(__func__, __builtin_return_address(0), __builtin_frame_address(0), alignment, size);
}

void *
malloc(size_t size)
{
	init();

	if (!__enable_dump_malloc && real_malloc)
		return real_malloc(size);

	return addtrack(__func__, __builtin_return_address(0), __builtin_frame_address(0), 1, size);
}

void *
calloc(size_t nmemb, size_t size)
{
	init();

	if (!__enable_dump_malloc && real_calloc)
		return real_calloc(nmemb, size);

	return addtrack(__func__, __builtin_return_address(0), __builtin_frame_address(0), 1, size * nmemb);
}

void *
realloc(void *ptr, size_t size)
{
	init();

	if (!__enable_dump_malloc && real_realloc)
		return real_realloc(ptr, size);

	Track *t = lookuptrack(ptr);
	void *p = malloc(size);
	if (t != NULL) {
		xprintf("frame=%p ret=%p realloc(old=%p oldsize=%zu new=%p newsize=%zu)\n",
		        __builtin_frame_address(0), __builtin_return_address(0), t->addr, t->size, p, size);
		memmove(p, t->addr, min(size, t->size));
	}
	return p;
}

void
free(void *ptr)
{
	init();

	if (!__enable_dump_malloc && real_free)
		return real_free(ptr);

	Track *t = lookuptrack(ptr);
	xprintf("free(frame=%p ret=%p addr=%p)%s",
	        __builtin_frame_address(0), __builtin_return_address(0), ptr,
	        (t == NULL) ? " invalid" : "");
}

#define SEP(x) ((x) == '/' || (x) == 0)

static char *
cleanname(char *name)
{
	char *s;  /* source of copy */
	char *d;  /* destination of copy */
	char *d0; /* start of path afer the root name */

	int rooted;
	if (name[0] == 0)
		return strcpy(name, ".");
	rooted = 0;
	d0 = name;
	if (d0[0] == '#') {
		if (d0[1] == 0)
			return d0;
		d0 += 1 + 1;
		while (!SEP(*d0)) {
			d0++;
		}
		if (d0 == 0)
			return name;
		d0++; /* keep / after #<name> */
		rooted = 1;
	} else if (d0[0] == '/') {
		rooted = 1;
		d0++;
	}
	s = d0;
	if (rooted) {
		/* skip extra '/' at root name */
		for (; *s == '/'; s++)
			;
	}
	/* remove dup slashes */
	for (d = d0; *s != 0; s++) {
		*d++ = *s;
		if (*s == '/')
			while (s[1] == '/')
				s++;
	}
	*d = 0;
	d = d0;
	s = d0;
	while (*s != 0) {
		if (s[0] == '.' && SEP(s[1])) {
			if (s[1] == 0)
				break;
			s += 2;
			continue;
		}
		if (s[0] == '.' && s[1] == '.' && SEP(s[2])) {
			if (d == d0) {
				if (rooted) {
					/* /../x -> /x */
					if (s[2] == 0)
						break;
					s += 3;
					continue;
				} else {
					/* ../x -> ../x; and never collect ../ */
					d0 += 3;
				}
			}
			if (d > d0) {
				/* a/../x -> x */
				assert(d - 2 >= d0 && d[-1] == '/');
				for (d -= 2; d > d0 && d[-1] != '/'; d--)
					;
				if (s[2] == 0)
					break;
				s += 3;
				continue;
			}
		}
		while (!SEP(*s))
			*d++ = *s++;
		if (*s == 0)
			break;
		*d++ = *s++;
	}
	*d = 0;
	if (d - 1 > name && d[-1] == '/') /* thanks to #/ */
		*--d = 0;
	if (name[0] == 0)
		strcpy(name, ".");
	return name;
}

static int
mkdirp(const char *dir)
{
	static _Thread_local char buf[PATH_MAX];

	char *p, *s;
	size_t n;
	int save_errno;

	save_errno = errno;
	snprintf(buf, sizeof(buf), "%s", dir);
	p = buf;
	if (!p) {
		errno = save_errno;
		return -1;
	}

	cleanname(p);
	n = strlen(p);
	if (p[n - 1] == '/')
		p[n - 1] = '\0';

	for (s = p + 1; *s; s++) {
		if (*s == '/') {
			*s = '\0';
			mkdir(p, S_IRWXU);
			*s = '/';
		}
	}
	mkdir(p, S_IRWXU);

	errno = save_errno;
	return 0;
}

static char *
rpath(const char *path)
{
	static _Thread_local char buf[PATH_MAX];
	snprintf(buf, sizeof(buf), "%s/%s", rootdir, path);
	return buf;
}
