#include <stdio.h>
#include <unistd.h>

int
add(int x, int y)
{
	return x + y;
}

int
runcode(void)
{
	const char s[] = "Running code\n";
	int x, y, z;
	int i;

	for (i = 0; i < 10; i++)
		write(1, s, sizeof(s));
	x = 2;
	y = 3;
	z = add(x, y);
	return z;
}

int
main(void)
{
	runcode();
	return 0;
}
