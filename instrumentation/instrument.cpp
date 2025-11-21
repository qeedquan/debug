#include <cstdio>
#include <dlfcn.h>

#ifdef __cplusplus
extern "C" {
#endif

__attribute__((no_instrument_function)) char *funcname(void *func)
{
	static thread_local char buf[128];

	Dl_info dlinfo;

	snprintf(buf, sizeof(buf), "?");
	if (dladdr(func, &dlinfo))
		snprintf(buf, sizeof(buf), "[%s] %s", dlinfo.dli_fname, dlinfo.dli_sname);

	return buf;
}

static thread_local unsigned indent;

__attribute__((no_instrument_function)) void __cyg_profile_func_enter(void *func, void *call)
{
	for (unsigned i = 0; i < indent; i++)
		printf("\t");
	printf("->%s\n", funcname(func));
	indent++;
}

__attribute__((no_instrument_function)) void __cyg_profile_func_exit(void *func, void *call)
{
	indent--;
	for (unsigned i = 0; i < indent; i++)
		printf("\t");
	printf("<-%s\n", funcname(func));
}

#ifdef __cplusplus
}
#endif

class Foo
{
public:
	void foo()
	{
		bar();
	}

	void bar()
	{
		baz();
	}

	static void baz()
	{
	}
};

int file(float x)
{
	return x;
}

int file(const char *s)
{
	return 0;
}

template<typename T>
void K(T t)
{
}

int main()
{
	Foo f;
	Foo::baz();
	f.foo();
	f.bar();
	file(100.0f);
	file("hello");
	K<int>(20);
	K<float>(40.0f);
	return 0;
}
