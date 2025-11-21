#include <stdlib.h>

int
main(void)
{
	char *p = malloc(10);
	for (int i = 0; i < 10; i++) {
		p[i] = '0' + i;
	}
	p = realloc(p, 1000);
	for (int i = 0; i < 1000; i++) {
		p[i] = i ^ 0xff;
	}
	p = calloc(60, 50);
	malloc(0);
	calloc(0, 0);
	free(p);
	free(p);
	free((void *)0xdeadbeef);
	return 0;
}
