#include <cstdio>

// class with no inheritance
class Foo
{
public:
	// compiles to a mangled function
	void kazz()
	{
		printf("%s\n", __func__);
	}

	void fset()
	{
		bar = baz = 405;
	}

	// compiles to a mangled function
	~Foo()
	{
		printf("Destructor\n");
	}

	// these will compile to the first member of the struct
	int bar;
	int baz;
};

class FooExtender : public Foo
{
public:
	FooExtender()
	{
		printf("Extender\n");
	}

	~FooExtender()
	{
		printf("%s\n", __func__);
	}

	void set()
	{
		x = 10;
		y = 20;
		z = 30;
		w = 40;
	}

	// these are just after a pointer so this would look like
	// struct FooExtender { Foo *foo; int x, y, z, w; }
	int x, y, z, w;
};

int main()
{
	Foo f;
	FooExtender fe;

	f.kazz();
	f.fset();
	fe.set();

	// this doesn't use any variable, it will just compile down to a function call
	fe.kazz();

	// this passes the foo pointer in the beginning of the class to it
	fe.fset();

	return 0;
}
