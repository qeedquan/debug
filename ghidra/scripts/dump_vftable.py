# Dump MSVC generated vftable structures that contains all of the function pointers for a class
#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar

import sys
import json

class Encoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, Vtable):
            return obj.serialize()
        return json.JSONEncoder.default(self, obj)

class Vtable:
    def __init__(self):
        self.name = ""
        self.cname = ""
        self.path = ""
        self.funcs = []

    def serialize(self):
        m = {}
        m["name"] = self.name
        m["cname"] = self.cname
        m["path"] = list(self.path)
        m["funcs"] = []
        for f in self.funcs:
            o = {}
            o["name"] = f.name
            o["cname"] = f.cname
            o["entry"] = f.entry.getOffset()
            o["exit"] = f.body.getMaxAddress().getOffset()
            o["sig"] = f.sig
            o["image"] = f.image.getOffset()
            m["funcs"].append(o)
        return m

class Func:
    def __init__(self):
        self.name = ""
        self.cname = ""
        self.body = None
        self.entry = 0
        self.exit = 0
        self.sig = ""
        self.image = 0

def make_c_name(name):
    cname = ""
    for c in name:
        if ('a' <= c and c <= 'z') or ('A' <= c and c <= 'Z') or ('0' <= c and c <= '9') or c == '_':
            cname += c
        else:
            cname += "_"
    return cname

def get_funcs(symbol):
    program = getCurrentProgram()
    funcmgr = program.getFunctionManager()
    addr = symbol.getAddress()
    data = getDataAt(addr)

    funcs = []
    seen = {}
    qid = 0
    for i in range(data.getNumComponents()):
        comp = data.getComponent(i)
        value = comp.getValue()
        func = funcmgr.getReferencedFunction(value)
        name = func.getName()
        cname = make_c_name(name)
        body = func.getBody()
        entry = func.getEntryPoint()
        sig = str(func.getSignature())
        if cname in seen:
            cname += str(qid)
            qid += 1
        seen[cname] = True

        f = Func()
        f.name = name
        f.cname = cname
        f.body = body
        f.entry = entry
        f.image = addr
        f.sig = sig
        funcs.append(f)
    return funcs

def get_vtables(symbol_name):
    program = getCurrentProgram()
    symbol_table = program.getSymbolTable()
    
    vtables = []
    seen = {}
    qid = 0
    for symbol in symbol_table.getSymbols(symbol_name):
        funcs = get_funcs(symbol)
        if len(funcs) == 0:
            continue

        path = symbol.getPath()
        name = "_".join(path)
        cname = make_c_name(name)
        if cname in seen:
            cname += str(qid)
            qid += 1
        seen[cname] = True

        v = Vtable()
        v.path = path
        v.name = name
        v.cname = cname
        v.funcs = funcs
        vtables.append(v)
    return vtables

def dump_c(stream, vtables):
    program = getCurrentProgram()
    stream.write("// Number of vtables: %d\n" % (len(vtables)))
    stream.write("// Image Base: %#x\n" % (program.getImageBase().getOffset()))

    for v in vtables:
        stream.write("typedef struct %s %s;\n" % (v.cname, v.cname))
    stream.write("\n")

    for v in vtables:
        stream.write("struct %s {\n" % (v.cname))
        i = 0
        for f in v.funcs:
            stream.write("\tvoid (*%s)(); // structure offset: %d image offset %#x: entry: %#x exit: %#x %s\n" %
                            (f.cname, i, f.image.getOffset(), f.entry.getOffset(), f.body.getMaxAddress().getOffset(), f.sig))
            i += 1
        stream.write("};\n")

def dump_vtables(stream, kind):
    vtables = get_vtables("vftable")
    if kind == "c":
        dump_c(stream, vtables)
    elif kind == "json":
        json.dump(vtables, stream, sort_keys=True, cls=Encoder, indent=4)
    else:
        raise Exception("Unknown output type %s" % (kind))

dump_vtables(sys.stdout, "json")
