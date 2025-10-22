#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdio.h>
#include <threads.h>

__attribute__((no_instrument_function)) char *
funcname(void *func)
{
	static thread_local char buf[128];

	Dl_info dlinfo;

	snprintf(buf, sizeof(buf), "?");
	if (dladdr(func, &dlinfo))
		snprintf(buf, sizeof(buf), "[%s] %s", dlinfo.dli_fname, dlinfo.dli_sname);

	return buf;
}

static thread_local unsigned indent;

__attribute__((no_instrument_function)) void
__cyg_profile_func_enter(void *func, void *call)
{
	for (unsigned i = 0; i < indent; i++)
		printf("\t");
	printf("->%s\n", funcname(func));
	indent++;
}

__attribute__((no_instrument_function)) void
__cyg_profile_func_exit(void *func, void *call)
{
	indent--;
	for (unsigned i = 0; i < indent; i++)
		printf("\t");
	printf("<-%s\n", funcname(func));
}

int
factorial(int n)
{
	if (n <= 1)
		return 1;
	return n * factorial(n - 1);
}

void
hal(int x)
{
}

int
main(void)
{
	factorial(10);
	for (int i = 0; i < 100; i++)
		hal(i);
	return 0;
}
