#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <inttypes.h>
#include <errno.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <unistd.h>
#include <dlfcn.h>
#include <pthread.h>

#define ADDR ((void *)0x5000000)
#define SIZE 0x80000000

#define nelem(x) (sizeof(x) / sizeof(x[0]))

typedef struct Arena Arena;
typedef struct Alloc Alloc;

struct Arena {
	pthread_mutex_t mutex;
	void *mem;
	size_t len;
	size_t cap;
	Alloc *use;
	Alloc *free;
};

struct Alloc {
	void *ret[8];
	void *frame[8];
	void *addr;
	uintptr_t start;
	uintptr_t end;
	size_t size;
	size_t alignment;
	Alloc *next;
	Alloc *tombs;
};

static void *(*real_malloc)(size_t);
static void *(*real_realloc)(void *, size_t);
static void *(*real_calloc)(size_t, size_t);
static void *(*real_memalign)(size_t, size_t);
static void (*real_free)(void *);

static Arena *arena;
static int phase;

bool __enable_custom_malloc = true;

#define xprintf(...)                                     \
	do {                                             \
		char buf[256];                           \
		snprintf(buf, sizeof(buf), __VA_ARGS__); \
		write(1, buf, strlen(buf));              \
	} while (0);

static void
freearena(Arena *a)
{
	if (a == NULL)
		return;
	if (a->cap > 0)
		munmap(a->mem, a->cap);
	free(a);
}

static Arena *
newarena(void *addr, size_t size)
{
	Arena *a = real_calloc(1, sizeof(*a));
	if (a == NULL)
		goto error;

	a->mem = mmap(addr, size, PROT_READ | PROT_WRITE | PROT_EXEC,
	              MAP_SHARED | MAP_ANONYMOUS | MAP_FIXED | MAP_NORESERVE, -1, 0);
	if (a->mem == MAP_FAILED)
		goto error;
	a->cap = size;

	pthread_mutex_init(&a->mutex, NULL);

	return a;

error:
	freearena(a);
	return NULL;
}

static void *
xgrow(Arena *a, Alloc **ac, void *optr, size_t alignment, size_t nmemb, size_t size)
{
	size_t len;
	if (__builtin_mul_overflow(nmemb, size, &len))
		return NULL;
	if (__builtin_add_overflow(len, 0x3full, &len))
		return NULL;
	len &= ~0x3full;

	if (phase == 1)
		return sbrk(len);

	Alloc *p = NULL;
	void *m = NULL;

	pthread_mutex_lock(&a->mutex);
	if (len >= a->cap || a->cap - len <= a->len)
		goto out;

	p = real_calloc(1, sizeof(*p));
	if (p == NULL)
		goto out;

	p->start = (uintptr_t)a->mem + a->len;
	p->size = size;
	p->end = p->start + size;
	p->alignment = alignment;
	p->addr = (void *)p->start;
	memset(p->addr, 0, p->size);
	if (optr)
		memmove(p->addr, optr, size);

	a->len += len;
	if (a->use == NULL)
		a->use = p;
	else {
		p->next = a->use;
		a->use = p;
	}

	if (ac)
		*ac = p;

	p->ret[0] = __builtin_return_address(0);
	p->ret[0] = __builtin_extract_return_addr(p->ret[0]);
	p->frame[0] = __builtin_frame_address(0);

out:
	if (p)
		m = p->addr;
	pthread_mutex_unlock(&a->mutex);
	return m;
}

static void *
xalloc(Arena *a, size_t alignment, size_t nmemb, size_t size)
{
	return xgrow(a, NULL, NULL, alignment, nmemb, size);
}

static void *
xrealloc(Arena *a, void *ptr, size_t size)
{
	Alloc *ac;
	void *m = xgrow(a, &ac, ptr, 1, 1, size);
	if (m == NULL)
		return NULL;

	pthread_mutex_lock(&a->mutex);
	Alloc *lp = NULL, *p = a->use;
	while (p != NULL) {
		if (p->addr == ptr) {
			if (lp != NULL)
				lp->next = p->next;

			Alloc *t = ac->tombs;
			if (t == NULL)
				ac->tombs = p;
			else {
				while (t->next != NULL)
					t = t->tombs;
				t->tombs = p;
			}

			break;
		}
		lp = p;
		p = p->next;
	}

	pthread_mutex_unlock(&a->mutex);

	return m;
}

static void
xfree(Arena *a, void *ptr)
{
	if (a == NULL)
		return;
	pthread_mutex_lock(&a->mutex);

	Alloc *lp = NULL, *p = a->use;
	while (p != NULL) {
		if (p->addr == ptr) {
			if (lp != NULL)
				lp->next = p->next;

			if (a->use == p)
				a->use = p->next;

			p->next = NULL;
			if (a->free == NULL)
				a->free = p;
			else {
				p->next = a->free;
				a->free = p;
			}

			break;
		}

		lp = p;
		p = p->next;
	}

	pthread_mutex_unlock(&a->mutex);
}

static void
init(void)
{
	if (phase >= 1)
		return;

	if (++phase != 1)
		return;

	real_malloc = dlsym(RTLD_NEXT, "malloc");
	real_realloc = dlsym(RTLD_NEXT, "realloc");
	real_calloc = dlsym(RTLD_NEXT, "calloc");
	real_memalign = dlsym(RTLD_NEXT, "memalign");
	real_free = dlsym(RTLD_NEXT, "free");

	arena = newarena(ADDR, SIZE);
	if (!arena) {
		xprintf("failed to make arena: %s\n", strerror(errno));
		_exit(1);
	}

	phase++;
}

void *
malloc(size_t size)
{
	init();
	if (!__enable_custom_malloc)
		return real_malloc(size);
	void *p = xalloc(arena, 1, 1, size);
	if (p == NULL)
		__builtin_trap();
	xprintf("malloc %p %zu\n", p, size);
	return p;
}

void *
calloc(size_t nmemb, size_t size)
{
	init();
	if (!__enable_custom_malloc)
		return real_calloc(nmemb, size);
	void *p = xalloc(arena, 1, nmemb, size);
	if (p == NULL)
		__builtin_trap();
	xprintf("calloc %p %zu\n", p, size);
	return p;
}

void *
memalign(size_t alignment, size_t size)
{
	init();
	if (!__enable_custom_malloc)
		return real_memalign(alignment, size);
	void *p = xalloc(arena, alignment, 1, size);
	if (p == NULL)
		__builtin_trap();
	xprintf("memalign %p %zu %zu\n", p, alignment, size);
	return p;
}

void *
realloc(void *ptr, size_t size)
{
	init();
	if (!__enable_custom_malloc)
		return realloc(ptr, size);

	void *p = xrealloc(arena, ptr, size);
	if (p == NULL)
		__builtin_trap();
	xprintf("realloc %p %p %zu\n", p, ptr, size);
	return p;
}

void
free(void *ptr)
{
	init();
	if (!__enable_custom_malloc)
		return;
	xfree(arena, ptr);
}

static void
dumplist(Alloc *p, const char *name)
{
	xprintf("\n%s\n", name);
	for (; p != NULL; p = p->next) {
		xprintf("frame %p ret %p addr %" PRIxPTR "-%" PRIxPTR " size %zu\n",
		        p->frame[0], p->ret[0], p->start, p->end, p->size);
	}
	xprintf("\n");
}

void
__dump_alloc_info(void)
{
	Arena *a = arena;

	pthread_mutex_lock(&a->mutex);

	dumplist(a->use, "Allocations");
	dumplist(a->free, "Frees");

	pthread_mutex_unlock(&a->mutex);
}
