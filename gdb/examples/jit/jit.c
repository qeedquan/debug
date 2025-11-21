/*

Example to show how to use GDB JIT interface.
1. Load the executable "code" into memory,
2. Register the debug information using the JIT interface.
3. Now when using GDB, we can see the debugging information when we call a function dynamically

*/

#include <elf.h>
#include <errno.h>
#include <fcntl.h>
#include <link.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <unistd.h>

/*

Use linker to fix the compiled code into a fixed address
The hack offset is to load the .text at the right place, skipping the ELF data that is not .text
The right way is to parse the ELF sections and figure out where the text section is and load just that into memory

*/

#define LOAD_ADDRESS (0x60000000 - 0x3000)

#include <stdint.h>

typedef enum {
	JIT_NOACTION = 0,
	JIT_REGISTER,
	JIT_UNREGISTER
} jit_actions_t;

struct jit_code_entry {
	struct jit_code_entry *next_entry;
	struct jit_code_entry *prev_entry;
	const void *symfile_addr;
	uint64_t symfile_size;
};

struct jit_descriptor {
	uint32_t version;
	uint32_t action_flag;
	struct jit_code_entry *relevant_entry;
	struct jit_code_entry *first_entry;
};

struct jit_descriptor __jit_debug_descriptor = {1, 0, 0, 0};

void __attribute__((noinline)) __jit_debug_register_code()
{
}

static void *
load_symbol(void *addr, const char *sym_name)
{
	const ElfW(Ehdr) *const ehdr = (ElfW(Ehdr) *)addr;
	ElfW(Shdr) *const shdr = (ElfW(Shdr) *)((char *)addr + ehdr->e_shoff);

	/* Find `func_name` in symbol_table and return its address.  */
	int i;
	for (i = 0; i < ehdr->e_shnum; ++i) {
		if (shdr[i].sh_type == SHT_SYMTAB) {
			ElfW(Sym) *symtab = (ElfW(Sym) *)((uintptr_t)addr + shdr[i].sh_offset);
			ElfW(Sym) *symtab_end = (ElfW(Sym) *)((uintptr_t)addr + shdr[i].sh_offset + shdr[i].sh_size);
			char *const strtab = (char *)((uintptr_t)addr + shdr[shdr[i].sh_link].sh_offset);

			ElfW(Sym) * p;
			for (p = symtab; p < symtab_end; ++p) {
				const char *s = strtab + p->st_name;
				if (strcmp(s, sym_name) == 0)
					return (void *)p->st_value;
			}
		}
	}

	fprintf(stderr, "symbol '%s' not found\n", sym_name);
	exit(1);
	return 0;
}

static void *
load_elf(const char *libname, size_t *size, void *load_addr)
{
	int fd;
	struct stat st;

	if ((fd = open(libname, O_RDONLY)) == -1) {
		fprintf(stderr, "open (\"%s\", O_RDONLY): %s\n", libname,
		        strerror(errno));
		exit(1);
	}

	if (fstat(fd, &st) != 0) {
		fprintf(stderr, "fstat (\"%d\"): %s\n", fd, strerror(errno));
		exit(1);
	}

	int pagesz = getpagesize();
	void *addr = mmap(load_addr, (st.st_size + (pagesz - 1)) & ~(pagesz - 1),
	                  PROT_READ | PROT_WRITE | PROT_EXEC,
	                  load_addr != NULL ? MAP_PRIVATE | MAP_FIXED : MAP_PRIVATE,
	                  fd, 0);
	close(fd);

	if (addr == MAP_FAILED) {
		fprintf(stderr, "mmap: %s\n", strerror(errno));
		exit(1);
	}

	if (size != NULL)
		*size = st.st_size;

	return addr;
}

void
hexdump(void *buf, size_t len)
{
	size_t i;
	uint8_t *ptr;

	ptr = buf;
	for (i = 0; i < len; i++)
		printf("%02x ", ptr[i]);
	printf("\n");
}

void
register_jit(void *addr, size_t obj_size)
{
	/* Link entry at the end of the list.  */
	struct jit_code_entry *const entry = calloc(1, sizeof(*entry));
	entry->symfile_addr = (const char *)addr;
	entry->symfile_size = obj_size;
	entry->prev_entry = __jit_debug_descriptor.relevant_entry;
	__jit_debug_descriptor.relevant_entry = entry;

	if (entry->prev_entry != NULL)
		entry->prev_entry->next_entry = entry;
	else
		__jit_debug_descriptor.first_entry = entry;

	/* Notify GDB.  */
	__jit_debug_descriptor.action_flag = JIT_REGISTER;
	__jit_debug_register_code();
}

int
main(void)
{
	void *addr;
	size_t size;
	int (*runfn)(void);
	int (*addfn)(int, int);

	addr = load_elf("code", &size, (void *)LOAD_ADDRESS);
	runfn = load_symbol(addr, "runcode");
	addfn = load_symbol(addr, "add");

	register_jit(addr, size);

	printf("runcode: %p\n", runfn);
	int ret = runfn();
	printf("return code: %d\n", ret);

	ret = addfn(5, 3);
	printf("add: %d\n", ret);

	return 0;
}
