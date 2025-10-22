# source gdb.py to use it
# py <defined_function> to call a python function

import ctypes
import gdb
import json
import string
import struct
import sys
import threading
import time
import os

wordfmts = [
    ["B", "u8", 1, "any", "unsigned"],
    ["<H", "u16le", 2, "little", "unsigned"],
    [">H", "u16be", 2, "big", "unsigned"],
    ["<I", "u32le", 4, "little", "unsigned"],
    [">I", "u32be", 4, "big", "unsigned"],
    ["<Q", "u64le", 8, "little", "unsigned"],
    [">Q", "u64be", 8, "big", "unsigned"],

    ["b", "s8", 1, "any", "signed"],
    ["<h", "s16le", 2, "little", "signed"],
    [">h", "s16be", 2, "big", "signed"],
    ["<i", "s32le", 4, "little", "signed"],
    [">i", "s32be", 4, "big", "signed"],
    ["<q", "s64le", 8, "little", "signed"],
    [">q", "s64be", 8, "big", "signed"],

    ["f", "f32", 4, "any", "float"],
    ["d", "f64", 8, "any", "float"],
]

class Timer(threading.Thread):
    def __init__(self, iterations, delay, action):
        threading.Thread.__init__(self)
        self.index = 0
        self.iterations = iterations
        self.delay = delay
        self.action = action 

    def run(self):
        while True:
            if self.iterations > 0 and self.index >= self.iterations:
                break

            time.sleep(self.delay)
            gdb.post_event(self.action)
            self.index += 1

class Object:
    pass

class Record:
    def __init__(self):
        self.addr = 0
        self.name = ""
        self.value = None
        self.backtrace = None

    def __str__(self):
        return "(%#x | %s)" % (self.addr, self.name)

    def __repr__(self):
        return self.__str__()

class Watch(gdb.Breakpoint):
    def __init__(self, spec, action, track=-1):
        if spec != "":
            gdb.Breakpoint.__init__(self, spec, type=gdb.BP_WATCHPOINT)

        self.action = action
        self.track = track
        self.target = None
        self.records = []
        self.hit = 0

        self.bp = []

    def event(self, ev):
        if isinstance(ev, gdb.BreakpointEvent):
            self.bp.append(ev.breakpoint.location)

    def stop(self):
        self.hit += 1
        if self.track > 0:
            if self.hit < self.track:
                return False

            t = parse_and_eval(self.expression)
            if self.target == None:
                self.target = t
            elif self.target != t:
                return False

        r = Record()
        r.addr = gdb.selected_frame().pc()
        r.name = self.expression
        r.value = self.action.evaluate(r.addr, r.name)
        self.records.append(r)
        return False

# Memory map range (start-end)
class Map:
    def __init__(self):
        self.kind = ""
        self.start = 0
        self.end = 0
        self.size = 0
        self.fileoff = 0
        self.perms = 0
        self.objfile = ""

    def __repr__(self):
        return "%#x-%#x %#x %#x %s %s" % (self.start, self.end, self.size, self.fileoff, self.objfile, self.kind)

# A symbol in memory (where it is in memory, size, what object file it belongs to/etc)
class Symbol:
    def __init__(self):
        self.exist = False
        self.kind = ""
        self.name = ""
        self.addr = 0
        self.saddr = 0
        self.eaddr = 0
        self.mapsz = 0 
        self.floff = 0 
        self.objfile = ""

# A database manages a set of memory maps
class Database:
    def __init__(self):
        self.maps = []
        self.vtables = []

    def add_vtable_file(self, filename):
        file = open(filename)
        vtable = json.load(file)
        file.close()
        self.vtables.append(vtable)

    # a map file is represented as lines of nm output format
    def add_map_file(self, filename):
        file = open(filename)
        data = file.read()
        lines = data.split("\n")
        file.close()
        self.maps.append(self.parse_map(lines))
        return self.maps[-1]

    def parse_map(self, lines):
        mp = []
        for line in lines:
            toks = line.split()
            if len(toks) < 4:
                continue
            try:
                m = Map()
                m.start = int(toks[0], 0)
                m.end = int(toks[1], 0)
                m.size = int(toks[2], 0)
                m.fileoff = int(toks[3], 0)
                m.perms = 0
                if len(toks) > 4:
                    end = len(toks)
                    if toks[end-1] == "T" or toks[end-1] == "D":
                        m.kind = toks[end-1]
                        end -= 1

                    m.objfile = "".join(toks[5:end])
                mp.append(m)
            except:
                pass
        return mp

    # get the current memory mapping of the process
    def get_maps(self):
        maps = self.maps.copy()
        try:
            procmaps = gdb.execute("info proc mappings", False, True)
            lines = procmaps.split('\n')
            maps.append(self.parse_map(lines))
        except:
            pass

        return maps

    # lookup a symbol in the memory map
    def lookup(self, value):
        s = Symbol()
        try:
            s.addr = int(parse_and_eval(value))
        except:
            s.addr = -1
            s.name = value

        fsz = 0
        for maps in self.get_maps():
            for mp in maps:
                if (s.addr < 0 and s.name == mp.objfile) or (mp.start <= s.addr and s.addr <= mp.end):
                    mpfsz = mp.end - mp.start + 1
                    if fsz == 0 or fsz > mpfsz:
                        fsz = mpfsz
                    else:
                        continue

                    s.kind = mp.kind
                    s.saddr = mp.start
                    s.eaddr = mp.end
                    s.mapsz = mp.size
                    s.floff = mp.fileoff
                    s.objfile = mp.objfile
                    if s.addr < 0:
                        s.addr = mp.start
                    if s.name == "":
                        s.name = s.objfile
                    s.exist = True
        return s

class Struct:
    def __init__(self):
        self.types = None
        self.endian = ""
        self.arch = ""
        self.nofollow = []

    def load_database(self, name):
        file = open(name)
        self.types = json.load(file)
        file.close()

        if self.arch != "":
            self.set_arch(self.arch)

    def set_arch(self, arch):
        arches = ["native", "32le", "32be", "64le", "64be"]
        if arch not in arches:
            raise Exception("Unknown architecture")

        if arch == "native":
            types = {
                "char": { "flags": ["signed"], "size": ctypes.sizeof(ctypes.c_char()) },
                "short": { "flags": ["signed"], "size": ctypes.sizeof(ctypes.c_short()) },
                "int": { "flags": ["signed"], "size": ctypes.sizeof(ctypes.c_int()) },
                "long": { "flags": ["signed"], "size": ctypes.sizeof(ctypes.c_long()) },
                "llong": { "flags": ["signed"], "size": ctypes.sizeof(ctypes.c_longlong()) },

                "uchar": { "flags": ["unsigned"], "size": ctypes.sizeof(ctypes.c_ubyte()) },
                "ushort": { "flags": ["unsigned"], "size": ctypes.sizeof(ctypes.c_ushort()) },
                "uint": { "flags": ["unsigned"], "size": ctypes.sizeof(ctypes.c_uint()) },
                "ulong": { "flags": ["unsigned"], "size": ctypes.sizeof(ctypes.c_ulong()) },
                "ullong": { "flags": ["unsigned"], "size": ctypes.sizeof(ctypes.c_ulonglong()) },

                "size_t": { "flags": ["unsigned"], "size": ctypes.sizeof(ctypes.c_size_t) },
                "ssize_t": { "flags": ["signed"], "size": ctypes.sizeof(ctypes.c_ssize_t) },

                "pointer": { "flags": ["pointer"], "size": ctypes.sizeof(ctypes.c_void_p()) },
            }
            endian = sys.byteorder
        else:
            types = {
                "char": { "flags": ["signed"], "size": 1 },
                "short": { "flags": ["signed"], "size": 2 },
                "int": { "flags": ["signed"], "size": 4 },
                "long": { "flags": ["signed"], "size": 8 },
                "llong": { "flags": ["signed"], "size": 8 },

                "uchar": { "flags": ["unsigned"], "size": 1 },
                "ushort": { "flags": ["unsigned"], "size": 2 },
                "uint": { "flags": ["unsigned"], "size": 4 },
                "ulong": { "flags": ["unsigned"], "size": 8 },
                "ullong": { "flags": ["unsigned"], "size": 8 },

                "size_t": { "flags": ["unsigned"], "size": 8 },
                "ssize_t": { "flags": ["signed"], "size": 8 },

                "pointer": { "flags": ["pointer"], "size": 8, "ref": "" },
            }
            if "32" in arch:
                types["size_t"]["size"] = 4
                types["ssize_t"]["size"] = 4
                types["pointer"]["size"] = 4

            if "le" in arch:
                endian = "little"
            else:
                endian = "big"

        types["cstring"] = { "flags": ["cstring"] }

        for typename in types:
            self.types[typename] = types[typename]
        self.endian = endian
        self.arch = arch
        self.sync_types()

    def sync_types(self):
        seen = {
            "char": True, "short": True, "int": True, "long": True, "llong": True,
            "uchar": True, "ushort": True, "uint": True, "ulong": True, "ullong": True,
            "size_t": True, "ssize_t": True,
            "pointer": True,
            "cstring": True,
        }
        for key in self.types:
            self.compute_type_closure(seen, key)

    def compute_type_closure(self, seen, typename):
        if typename in seen:
            return

        ty = self.types[typename]
        if "typedef" in ty["flags"]:
            self.compute_type_closure(seen, ty["typedef"])
            ty["size"] = self.types[ty["typedef"]]["size"]
        elif "enum" in ty["flags"]:
            ty["size"] = self.types["int"]["size"]
        elif "pointer" in ty["flags"]:
            ty["size"] = self.types["pointer"]["size"]
        elif "array" in ty["flags"]:
            self.compute_type_closure(seen, ty["element"])
            ty["size"] = self.types[ty["element"]]["size"] * ty["length"]
        elif "struct" in ty["flags"] or "union" in ty["flags"]:
            size = 0
            maxsize = 0
            for mv in ty["members"]:
                self.compute_type_closure(seen, mv[1])
                size += self.types[mv[1]]["size"]
                maxsize = max(maxsize, self.types[mv[1]]["size"])

            if "struct" in ty["flags"]:
                ty["size"] = size
            else:
                ty["size"] = maxsize

            if ty["size"] == 0:
                ty["size"] = 1

        seen[typename] = True

    def read(self, typename, expr):
        addr = int(parse_and_eval(expr))
        links = {}
        return self.make_struct("", links, typename, addr)

    def get_underlying_size(self, typename):
        ty = self.types[typename]
        while "pointer" in ty["flags"]:
            ty = self.types[ty["ref"]]
        return ty["size"]

    def make_struct(self, path, links, typename, addr):
        if path != "":
            path += "."
        path += typename 

        ty = self.types[typename]
        if "typedef" in ty["flags"]:
            ty = self.types[ty["typedef"]]

        if "cstring" in ty["flags"]:
            buf = ""
            off = 0
            while True:
                val = read_mem(addr+off, 1)[0]
                if val == 0:
                    break
                off += 1
                buf += chr(val)
            obj = buf
        elif "struct" in ty["flags"]:
            obj = Object()
            off = 0
            for mv in ty["members"]:
                val = self.make_struct(path, links, mv[1], addr+off)
                off += self.types[mv[1]]["size"]
                setattr(obj, mv[0], val)
        elif "union" in ty["flags"]:
            obj = list(read_mem(addr, ty["size"]))
        elif "array" in ty["flags"]:
            obj = []
            for i in range(ty["length"]):
                val = self.make_struct(path, links, ty["element"], addr)
                obj.append(val)
                size = self.types[ty["element"]]["size"]
                addr += size
        else:
            try:
                data = read_mem(addr, ty["size"])
            except:
                data = b"\x00" * ty["size"]

            obj = data
            for fmt in wordfmts:
                isenum = "enum" in ty["flags"] and fmt[4] == "signed"
                isptr = "pointer" in ty["flags"] and fmt[4] == "unsigned"
                if fmt[2] == ty["size"] and (fmt[3] == self.endian or fmt[3] == "any") and (fmt[4] in ty["flags"] or isenum or isptr):
                    obj = struct.unpack_from(fmt[0], data)[0]
                    break

            if "pointer" in ty["flags"] and typename not in self.nofollow and self.get_underlying_size(typename) != 0:
                try:
                    obj = self.make_struct(path, links, ty["ref"], obj) 
                except:
                    pass

        links[path] = obj
        return obj

def perm2str(mode):
    buf = "rwx-"
    if mode&1 == 0:
        buf = buf.replace('r', '-')
    if mode&2 == 0:
        buf = buf.replace('w', '-')
    if mode&4 == 0:
        buf = buf.replace('x', '-')
    return buf

def parse_and_eval(expr):
    if isinstance(expr, int) or isinstance(expr, gdb.Value):
        expr = str(expr)
    return gdb.parse_and_eval(expr)

def jsonfmt(obj):
    if isinstance(obj, Object):
        return json.dumps(obj, default=lambda o: o.__dict__, indent=4)
    return json.dumps(obj, indent=4)

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

def mapto(expr, db=None):
    if db == None:
        db = Database()

    sym = db.lookup(parse_and_eval(expr))
    if sym.exist:
        print("name '%s' addr %#x range %#x-%#x size %#x mapsize %#x fileoff %#x objfile %s" % 
                (sym.name, sym.addr, sym.saddr, sym.eaddr, sym.addr-sym.saddr, sym.mapsz, sym.floff, sym.objfile))
    else:
        print("No mapping found for", value)

# dump memory in a xxd like style
# value can be a symbol
def xxd(value, size=256, column=16, base='x', db=None):
    if db == None:
        db = Database()

    s = db.lookup(value)
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

            name = ""
            s = db.lookup(addr + i)
            if s.exist:
                name = s.name
            print('%16s 0x%016x| ' % (name, addr+i), end='')

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

def read_mem(expr, size):
    addr = int(parse_and_eval(expr))
    f = gdb.inferiors()[0]
    m = f.read_memory(addr, size)
    return m.tobytes()

def read_until_terminator(expr, terminator):
    addr = int(parse_and_eval(expr))
    f = gdb.inferiors()[0]
    m = b""
    while True:
        b = f.read_memory(addr, len(terminator)).tobytes()
        if b == terminator:
            break
        m += b
        addr += len(terminator)
    return m

def read_utf16_cstring(expr):
    return read_until_terminator(expr, b"\x00\x00")

def read_utf16_cstring_as_ascii(expr):
    return utf16_cstring_to_ascii(read_utf16_cstring(expr))

def utf16_cstring_to_ascii(s):
    r = ""
    i = 0
    for c in s:
        if i%2 == 0:
            r += chr(c)
        i += 1
    return r

def find_mem(addr, size, value, cb=None):
    f = gdb.inferiors()[0]
    m = f.read_memory(addr, size)
    b = m.tobytes()
    
    i = 0
    while i < len(b):
        for fmt in wordfmts:
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

def find_bytes(start, end, pattern):
    size = end - start + 1
    f = gdb.inferiors()[0]
    m = f.read_memory(start, size)
    b = m.tobytes()

    p = start
    while p < end:
        l = b.find(pattern)
        if l < 0:
            break
        b = b[l+len(pattern):]
        p += l
        print("Found bytes at %#x %s" % (p, pattern.hex()))

def match_vtable(expr, ptrfmt, db, base, reloc):
    addr = int(parse_and_eval(expr))
    size = -1
    fmt = ""
    for f in wordfmts:
        if ptrfmt == f[1]:
            size = f[2]
            fmt = f[0]
            break

    if size < 0:
        raise Exception("Unknown pointer format")
    
    for t in db.vtables:
        count = 0
        for v in t:
            matches = []
            funcs = v["funcs"]            
            for i in range(len(funcs)):
                data = read_mem(addr + i*size, size)
                ptr = struct.unpack_from(fmt, data)[0]
                ptr = ptr - reloc + base
                entry = funcs[i]["entry"]
                if ptr == entry:
                    matches.append([i, funcs[i]["sig"]])

            if len(matches) > 0:
                count += 1
                print("VTABLE MATCH %d | %s %d/%d" % (count, v["name"], len(matches), len(funcs)))
                for m in matches:
                    print(m[0], m[1])
                print()
                
def btr(db, base, reloc):
    output = gdb.execute("bt", False, True)
    lines = output.split('\n')
    for line in lines:
        print(line, end='', flush=True)
        if "??" in line:
            try:
                toks = line.replace("  ", " ").split(" ")
                addr = int(toks[1], 0)
                sym = db.lookup(addr - reloc + base)
                if sym.exist:
                    print(" | addr %#x off %#x func %#x %s" % (sym.addr, sym.addr - sym.saddr, sym.saddr, sym.objfile), end='')
            except Exception as ex:
                print(" | %s" % (ex), end='')
        print()
    
    snap = snapshot()
    hexlist = ["{:#04x}".format(x) for x in range(256)]
    for reg in snap.reg:
        vals = []
        if reg in snap.mem:
            data = snap.mem[reg]
            length = min(len(data), 16)
            vals = ["0x%02X" % n for n in data[:length]]

        print("%-8s | %#16x | %s" % (reg, snap.reg[reg], vals))

def br(value, db, base, reloc, break_all):
    sym = db.lookup(value)
    start = sym.addr - base + reloc
    end = start
    if break_all:
        end = sym.eaddr - base + reloc
    
    if start == end:
        print("Breaking at '%s' | %#x -> %#x" % (value, sym.addr, start))
    else:
        print("Breaking at '%s' | %#x %#x -> %#x %#x" % (value, sym.addr, sym.eaddr, start, end))

    for addr in range(start, end+1):
        cmd = "b *%#x" % addr
        gdb.execute(cmd)

def get_call_frames(db, base, reloc):
    calls = []
    frame = gdb.newest_frame()
    while frame != None:
        addr = frame.pc()
        sym = db.lookup(addr - reloc + base)
        
        call = Record()
        call.addr = addr
        call.name = sym.name
        calls.append(call)
        
        frame = frame.older()
    return calls

def match_call_names(calls, names):
    length = min(len(calls), len(names))
    for i in range(length):
        if calls[i].name != names[i]:
            return False
    return True

def match_call_addrs(calls, addrs):
    length = min(len(calls), len(addrs))
    for i in range(length):
        if calls[i].addr != addrs[i]:
            return False
    return True

def snapshot(peek_bytes=64, regmap=None):
    snap = Object()
    snap.reg = {}
    snap.mem = {}

    output = gdb.execute('info registers', False, True)
    lines = output.split('\n')
    for line in lines:
        toks = line.split(" ")
        if len(toks) < 1 or toks[0] == "":
            continue
        
        reg = toks[0]
        val = int(gdb.parse_and_eval("$" + reg))
        snap.reg[reg] = val

        addr = val
        length = peek_bytes
        if regmap != None and reg in regmap:
            if "size" in regmap[reg]:
                length = regmap[reg]["size"]
            if "off" in regmap[reg]:
                addr += regmap[reg]["off"]
        
        if length <= 0:
            continue

        try:
            snap.mem[reg] = list(read_mem(addr, length))
        except:
            pass
    
    return snap

# dump target memory (memory mapped during program execution; this can be dynamic over the lifetime of the program)
def dump_target_memory_map(topdir, predfn=None):
    output = gdb.execute('info target', False, True)
    mappings = []
    for line in output.split('\n'):
        tokens = line.split(' ')
        if len(tokens) < 5:
            continue
        try:
            mp = Map()
            mp.start = int(tokens[0], 16)
            mp.end = int(tokens[2], 16)
            mp.size = mp.end - mp.start + 1
            mp.perms = 0o777
            mp.objfile = tokens[4]
            if len(tokens) >= 7:
                mp.objfile += " " + tokens[6]
            if predfn and predfn(mp):
                mappings.append(mp)
        except:
            pass
    dump_memory_map(topdir, mappings)

# given a list of mappings (array of Map), dump it to a to a directory
def dump_memory_map(topdir, mappings):
    os.makedirs(topdir, exist_ok=True)

    mapfile = os.path.join(topdir, "maps.txt")
    f = open(mapfile, "w")
    for mp in mappings:
        f.write("%#-20x %#-20x %#-12x %#-8x %s %s\n" % (mp.start, mp.end, mp.size, 0, perm2str(mp.perms), mp.objfile))
        try:
            name = os.path.join(topdir, "mem_%x_%x.bin" % (mp.start, mp.end))
            gdb.execute("dump binary memory %s %#x %#x" % (name, mp.start, mp.end))
        except:
            print("WARN: failed to dump %#x-%#x" % (mp.start, mp.end))
    f.close()

def funargs(kind, fmt):
    ABI = {
        "win64": {
            "int_regs":   ["$rcx", "$rdx", "$r8", "$r9"],
            "float_regs": ["$xmm0", "$xmm1", "$xmm2", "$xmm3"],
            "increment_mode": 1,
            "stack_offset": 0x30,
            "stack_argsize": 8,
            "pointer_size": 8,
        }
    }

    MASK = {
        'c': (1<<8) - 1,
        'h': (1<<16) - 1,
        'i': (1<<32) - 1,
        'l': (1<<64) - 1,
    }

    ai = 0
    af = 0
    abi = ABI[kind]
    args = []
    for c in fmt:
        val = ""
        if c == 'p':
            if abi["pointer_size"] == 8:
                c = 'l'
            else:
                c = 'i'
        elif c == 'x':
            c = 'i'
        elif c == 'z':
            c = 'l'

        if c in "chil":
            if ai < len(abi["int_regs"]):
                expr = abi["int_regs"][ai]
            else:
                expr = "*(long*)($rsp + %d)" % (abi["stack_offset"] + abi["stack_argsize"]*(ai-len(abi["int_regs"])))
            val = gdb.parse_and_eval(expr) & MASK[c]
        elif c == 'f':
            if af < len(abi["float_regs"]):
                expr = abi["float_regs"][af] + ".v4_float[0]"
            else:
                expr = "*(float*)($rsp + %d)" % (abi["stack_offset"] + abi["stack_argsize"]*(ai-len(abi["float_regs"])))
            val = gdb.parse_and_eval(expr)
        elif c == 'd':
            if af < len(abi["float_regs"]):
                expr = abi["float_regs"][af] + ".v2_double[0]"
            else:
                expr = "*(double*)($rsp + %d)" % (abi["stack_offset"] + abi["stack_argsize"]*(ai-len(abi["float_regs"])))
            val = gdb.parse_and_eval(expr)

        args.append(val)
        
        if abi["increment_mode"] == 1:
            ai += 1
            af += 1

    return args

def print_funargs(kind, fmt):
    args = funargs(kind, fmt)
    i = 0
    for c in fmt:
        print("Arg %d: " % (i), end='')
        if c in "pxz":
            print(hex(args[i]))
        elif c in "chil":
            print(int(args[i]))
        else:
            print(str(args[i]))
        i += 1
    
def trace_functions(iterations, suppression_threshold, callback):
    records = []
    count = {}
    while iterations != 0:
        gdb.execute("c")
        
        pc = gdb.selected_frame().pc()
        
        if pc not in count:
            count[pc] = 0

        if suppression_threshold > 0 and count[pc] > suppression_threshold:
            gdb.execute("clear *%#x" % pc)

        count[pc] += 1

        r = Record()
        r.addr = pc
        r.value = gdb.execute("i r", to_string=True)
        try:
            r.backtrace = gdb.execute("bt", to_string=True)
        except:
            pass

        if callback != None:
            callback(r, count)
        else:
            records.append(r)
        
        if iterations > 0:
            iterations -= 1

    return records
