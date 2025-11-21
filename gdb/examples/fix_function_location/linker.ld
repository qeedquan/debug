ENTRY(_start)

START = 0x140000000;

SECTIONS
{
	. = START;
	.text :
	{
		*(.text.main)
		BYTE(0xff);
		BYTE(0xff);
		*(.text.f2)
		. = ALIGN(0x800);
		*(.text.f1)
		. = ALIGN(0x1000);
		*(.text)
	}
	. = ALIGN(4096);

	.data :
	{
		*(.data)
	}
	. = ALIGN(4096);

	.bss :
	{
		__bss_start__ = . ;
		*(.bss)
		*(COMMON)
		__bss_end__ = . ;
	}
	. = ALIGN(4096);
}
