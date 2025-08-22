// This prints backtrace without source code location
#include <execinfo.h>
#include <stdio.h>
#include <stdlib.h>

/* Obtain a backtrace and print it to stdout. */
void
print_trace(void)
{
	void *array[10];
	char **strings;
	int size, i;

	size = backtrace(array, 10);
	strings = backtrace_symbols(array, size);
	if (strings != NULL) {

		printf("Obtained %d stack frames.\n", size);
		for (i = 0; i < size; i++)
			printf("%s\n", strings[i]);
	}

	free(strings);
}

/* A dummy function to make the backtrace more interesting. */
void
dummy_function(void)
{
	print_trace();
}

int
main(void)
{
	dummy_function();
	return 0;
}
