/*

Example of placing individual functions at a specific address

1. Use -ffunction-sections to make the compiler put each function into its own section
2. Inside the linker script, we can use these sections to layout to specific locations
3. For padding, use BYTE(x) directive, ALIGN(x) only works with powers of 2 addresses

*/

#include <stdio.h>
#include <stdlib.h>

int
f1(int x, int y, int z)
{
	return x + y + z;
}

int
f2(int x)
{
	return x * x;
}

int
main()
{
	int r;

	r = f1(1, 2, 3);
	r += f2(4);
	printf("%d\n", r);
	return r;
}
