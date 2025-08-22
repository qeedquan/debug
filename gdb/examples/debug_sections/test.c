/*

Example of reading a symbol file into GDB by:

1. Write all of the debugging symbols from a binary to a symbol file
2. Strip the binary of all debugging symbols
3. In GDB, use "symbol-file" on the symbol file written out to get back debugging information
*/

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
	return r;
}
