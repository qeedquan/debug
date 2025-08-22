// This prints a backtrace with source code location
#include <inttypes.h>
#include <stdio.h>
#include <unistd.h>
#include <execinfo.h>
#include <elfutils/libdwfl.h>

void
dump_frames(void)
{
	char *debuginfo_path = NULL;
	const Dwfl_Callbacks callbacks = {
		.find_debuginfo = dwfl_standard_find_debuginfo,
		.find_elf = dwfl_linux_proc_find_elf,
		.debuginfo_path = &debuginfo_path,
	};
	Dwfl *dwfl;
	void *frames[48];
	int n, n_ptrs;

	dwfl = dwfl_begin(&callbacks);

	if (dwfl_linux_proc_report(dwfl, getpid()))
		goto done;

	dwfl_report_end(dwfl, NULL, NULL);

	n_ptrs = backtrace(frames, 48);
	if (n_ptrs < 1)
		goto done;

	printf("++++++++ backtrace ++++++++\n");

	for (n = 1; n < n_ptrs; n++) {
		GElf_Addr addr = (uintptr_t)frames[n];
		GElf_Sym sym;
		GElf_Word shndx;
		Dwfl_Module *module = dwfl_addrmodule(dwfl, addr);
		Dwfl_Line *line;
		const char *name, *modname;

		if (!module) {
			printf("#%-2u ?? [%#" PRIx64 "]\n", n, addr);
			continue;
		}

		name = dwfl_module_addrsym(module, addr, &sym, &shndx);
		if (!name) {
			modname = dwfl_module_info(module, NULL, NULL, NULL,
			    NULL, NULL, NULL, NULL);
			printf("#%-2u ?? (%s) [%#" PRIx64 "]\n",
			    n, modname, addr);
			continue;
		}

		line = dwfl_module_getsrc(module, addr);
		if (line) {
			int lineno;
			const char *src = dwfl_lineinfo(line, NULL, &lineno,
			    NULL, NULL, NULL);

			if (src) {
				printf("#%-2u %s+%#" PRIx64 " "
				       "(%s:%d) [%#" PRIx64 "]\n",
				    n, name, addr - sym.st_value,
				    src, lineno, addr);
				continue;
			}
		}

		modname = dwfl_module_info(module, NULL, NULL, NULL,
		    NULL, NULL, NULL, NULL);
		printf("#%-2u %s+%#" PRIx64 " (%s) [%#" PRIx64 "]\n",
		    n, name, addr - sym.st_value,
		    modname, addr);
	}

	printf("+++++++++++++++++++++++++++\n");

done:
	dwfl_end(dwfl);
}

void
foo(int n)
{
	if (n <= 0) {
		dump_frames();
		return;
	}

	foo(n - 1);
}

int
main(void)
{
	void *frames[1];

	backtrace(frames, 1);
	foo(100);
	return 0;
}
