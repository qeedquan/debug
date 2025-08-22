// translate values into various byte encodings of various sized type
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <inttypes.h>

void
scan(const char *s, const char *t, const char *f, void *p, size_t n)
{
	char *v;
	size_t i;

	if (sscanf(s, f, p) != 1)
		return;

	v = p;
	printf("%-16s: ", t);
	for (i = 0; i < n; i++)
		printf("%02x ", v[i] & 0xff);
	printf("\n");
}

void
usage(void)
{
	fprintf(stderr, "usage: value ...\n");
	exit(2);
}

int
main(int argc, char *argv[])
{
	int i, j;
	char c;
	short s;
	long l;
	long long ll;
	unsigned u;
	int8_t i8;
	int16_t i16;
	int32_t i32;
	int64_t i64;
	intmax_t imax;
	uint8_t u8;
	uint16_t u16;
	uint32_t u32;
	uint64_t u64;
	uintmax_t umax;
	float f;
	double d;
	long double ld;

	if (argc < 2)
		usage();

	for (i = 1; i < argc; i++) {
		printf("%s\n", argv[i]);
		scan(argv[i], "char", "%hhi", &c, sizeof(c));
		scan(argv[i], "short", "%hi", &s, sizeof(s));
		scan(argv[i], "long", "%li", &l, sizeof(l));
		scan(argv[i], "long long", "%lli", &ll, sizeof(ll));
		scan(argv[i], "uint", "%u", &u, sizeof(u));
		scan(argv[i], "int", "%i", &j, sizeof(j));
		scan(argv[i], "int8", "%" SCNi8, &i8, sizeof(i8));
		scan(argv[i], "int16", "%" SCNi16, &i16, sizeof(i16));
		scan(argv[i], "int32", "%" SCNi32, &i32, sizeof(i32));
		scan(argv[i], "int64", "%" SCNi64, &i64, sizeof(i64));
		scan(argv[i], "intmax", "%" SCNiMAX, &imax, sizeof(imax));
		scan(argv[i], "uint8", "%" SCNu8, &u8, sizeof(u8));
		scan(argv[i], "uint16", "%" SCNu16, &u16, sizeof(u16));
		scan(argv[i], "uint32", "%" SCNu32, &u32, sizeof(u32));
		scan(argv[i], "uint64", "%" SCNu64, &u64, sizeof(u64));
		scan(argv[i], "uintmax", "%" SCNuMAX, &umax, sizeof(umax));
		scan(argv[i], "float", "%f", &f, sizeof(f));
		scan(argv[i], "double", "%lf", &d, sizeof(d));
		scan(argv[i], "long double", "%Lf", &ld, sizeof(ld));
		printf("\n");
	}
	return 0;
}
