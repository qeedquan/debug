# source gdb.py to use it
# py <defined_function> to call a python function
import gdb
import string
import struct

class Map:
    def __init__(self):
        self.start = 0
        self.end = 0
        self.size = 0
        self.fileoff = 0
        self.objfile = 0

class Symbol:
    def __init__(self):
        self.exist = False
        self.name = ''
        self.addr = 0
        self.saddr = 0
        self.eaddr = 0
        self.mapsz = 0 
        self.floff = 0 
        self.objsize = 0
        self.objfile = ''

    def lookup_runtime(self, value):
        self.exist = False
        if type(value) is str:
            self.addr = strint(gdb.execute('print &%s' % (value), False, True))
            self.name = value
        elif type(value) is int:
            buf = gdb.execute('info symbol %#x' % (value), False, True)
            if buf.startswith('No symbol'):
                self.name = ''
            else:
                self.name = buf.split()[0]
            self.addr = value

        maps = getmaps()
        for mp in maps:
            if mp.start <= self.addr and self.addr <= mp.end:
                self.saddr = mp.start
                self.eaddr = mp.end
                self.mapsz = mp.size
                self.floff = mp.fileoff
                self.exist = True
                break
        return self.exist

def getmaps():
    maps = []
    procmaps = gdb.execute("info proc mappings", False, True)
    lines = procmaps.split('\n')
    for line in lines:
        toks = line.split()
        if len(toks) < 4:
            continue
        try:
            mp = Map()
            mp.start = int(toks[0], 0)
            mp.end = int(toks[1], 0)
            mp.size = int(toks[2], 0)
            mp.fileoff = int(toks[3], 0)
            if len(toks) > 4:
                mp.objfile = "".join(toks[4:])
            maps.append(mp)
        except:
            pass
    return maps

def rumul(num, multiple):
    return ((num + multiple - 1) // multiple) * multiple

def strint(line):
    for tok in line.split():
        try:
            val = int(tok, 0)
            return val
        except:
            pass
    raise ValueError("Can't find integer in string")

def mapto(value):
    sym = Symbol()
    if sym.lookup_runtime(value):
        print('%s %#x %#x-%#x %#x %#x %#x %s' % (sym.name, sym.addr, sym.saddr, sym.eaddr, sym.addr-sym.saddr, sym.mapsz, sym.floff, sym.objfile))
    else:
        print("No mapping found for", value)

def xxd(value, size=256, column=16, base='x'):
    s = Symbol()
    s.lookup_runtime(value)
    addr = s.addr
    
    f = gdb.inferiors()[0]
    m = f.read_memory(addr, size)
    b = m.tobytes()

    for i in range(rumul(len(b)+1, column)+1):
        if i%column == 0:
            if i > 0:
                print(' |', end='')
                for j in range(column):
                    k = i - (column - j)
                    if k >= len(b):
                        print(' ', end='')
                    elif chr(b[k]) in string.printable and chr(b[k]) not in string.whitespace:
                        print('%c' % (b[k]), end='')
                    else:
                        print('.', end='')
                print()
            s = Symbol()
            if s.lookup_runtime(addr + i):
                print('%16s 0x%016x| ' % (s.name, addr+i), end='')

        if i%(column/2) == 0 and i%column != 0 and i > 0:
            print('  ', end='')
        
        if i < len(b):
            if base == 'd':
                print('%3d ' % (b[i]), end='')
            else:
                print('%02x ' % (b[i]), end='')
        else:
            print('   ', end='')
            if base != 'x':
                print(' ', end='')
    
    print()

def findmem(addr, size, value, cb=None):
    fmts = [
        ["b", "u8"],
        ["<h", "u16le"],
        [">h", "u16be"],
        ["<i", "u32le"],
        [">i", "u32le"],
        ["<q", "u64le"],
        [">q", "u64be"]
    ]

    f = gdb.inferiors()[0]
    m = f.read_memory(addr, size)
    b = m.tobytes()
    
    i = 0
    while i < len(b):
        for fmt in fmts:
            try:
                dp = struct.unpack_from(fmt[0], b, i)
                if cb != None:
                    cb(addr + i, fmt[1])
                elif value == dp[0]:
                    print("%#x: off %#x type %s val %d %d %#x %#x\n" % (addr + i, i, fmt[1], value, dp[0], value, dp[0]), end='')
            except:
                pass
        
        if i%0x1000000 == 0 and cb != None:
                print("finding memory at %#x...\n" % (addr + i), end='')
        i += 1

