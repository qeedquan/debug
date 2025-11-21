all:
	cc -fPIC -shared -o libdump_malloc.so dump_malloc.c -ggdb -g3 -Wall -ldl
	cc -fPIC -shared -o libdump_malloc_flat.so dump_malloc_flat.c -ggdb -g3 -Wall -ldl
	cc -fPIC -shared -o libdump_mmap.so dump_mmap.c -ggdb -g3 -Wall -ldl
	cc -o test test.c -Wall
	cc -o test_flat test_flat.c -Wall -L. -ldump_malloc_flat
	cc -o test_mmap test_mmap.c -Wall -L. -ldump_mmap
	LD_PRELOAD=$(PWD)/libdump_mmap.so ./test_mmap
	LD_PRELOAD=$(PWD)/libdump_malloc.so ./test
	LD_LIBRARY_PATH=$(LD_LIBRARY_PATH):$(PWD) LD_PRELOAD=$(PWD)/libdump_malloc_flat.so ./test_flat

clean:
	rm -f test test_flat test_mmap
	rm -f *.so
	rm -rf dump_malloc/
