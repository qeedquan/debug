#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

import sys
import json

class Encoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, Table):
            return obj.serialize()
        return json.JSONEncoder.default(self, obj)

class Table:
    def __init__(self):
        self.symbols = []

    def serialize(self):
        l = []
        for s in self.symbols:
            m = {}
            m["name"] = s.name
            m["path"] = s.path
            m["addr"] = s.addr
            m["location"] = s.location
            m["symbol_type"] = s.symbol_type
            m["source_type"] = s.source_type
            m["references"] = s.references
            if s.data != None:
                m["data"] = s.data
            l.append(m)
        return l

class Symbol:
    def __init__(self):
        self.name = ""
        self.path = []
        self.addr = None
        self.data = None
        self.location = None
        self.symbol_type = None
        self.source_type = None
        self.references = []

def dump_symbols(stream=sys.stdout):
    program = getCurrentProgram()
    symtab = program.getSymbolTable()
    t = Table()
    t.symbols = []
    for sym in symtab.getAllSymbols(True):
        name = sym.getName(False)
        addr = sym.getAddress()
        path = sym.getPath()
        references = sym.getReferences()
        data = getDataAt(addr)

        p = Symbol()
        p.name = sym.getName(False)
        p.addr = str(addr)
        p.symbol_type = str(sym.getSymbolType())
        p.source_type = str(sym.getSource())
        p.location = str(sym.getProgramLocation())

        if data != None:
            p.data = str(data)
        for i in range(len(path)):
            p.path.append(str(path[i]))
        for i in range(len(references)):
            p.references.append(str(references[i]))
        t.symbols.append(p)

    json.dump(t, stream, sort_keys=True, cls=Encoder, indent=4)

dump_symbols()
