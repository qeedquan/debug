#define _GNU_SOURCE
#include <stdio.h>
#include <malloc.h>
#include <stdlib.h>

void __dump_alloc_info();

int
main(void)
{
	malloc(100505);
	void *p = realloc(NULL, 405);
	free(p);
	__dump_alloc_info();
	return 0;
}
