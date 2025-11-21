#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

import json
import sys

class Encoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, Type):
            return obj.serialize()
        return json.JSONEncoder.default(self, obj)

class Type:
    def __init__(self):
        self.size = 0
        self.flags = []
        self.length = 0
        self.ref = None
        self.element = None
        self.members = None
        self.typedef = None

    def serialize(self):
        m = { 
            "size": self.size,
            "flags": self.flags,
        }
        
        if "pointer" in self.flags:
            m["ref"] = self.ref
        elif "array" in self.flags:
            m["element"] = self.element
            m["length"] = self.length
        elif "struct" in self.flags or "union" in self.flags:
            m["members"] = self.members

        if "typedef" in self.flags:
            m["typedef"] = self.typedef

        return m

def get_type_attribute(kind):
    if isinstance(kind, ghidra.program.database.data.PointerDB):
        return "pointer"
    if isinstance(kind, ghidra.program.database.data.ArrayDB):
        return "array"
    if isinstance(kind, ghidra.program.database.data.StructureDB):
        return "struct"
    if isinstance(kind, ghidra.program.database.data.UnionDB):
        return "union"
    if isinstance(kind, ghidra.program.database.data.EnumDB):
        return "enum"

    if isinstance(kind, ghidra.program.model.data.UnsignedCharDataType):
        return "unsigned"

    if isinstance(kind, ghidra.program.model.data.CharDataType):
        return "signed"
    if isinstance(kind, ghidra.program.model.data.ShortDataType):
        return "signed"
    if isinstance(kind, ghidra.program.model.data.LongDataType):
        return "signed"
    if isinstance(kind, ghidra.program.model.data.IntegerDataType):
        return "signed"

    if isinstance(kind, ghidra.program.model.data.FloatDataType):
        return "float"
    if isinstance(kind, ghidra.program.model.data.DoubleDataType):
        return "float"
    if isinstance(kind, ghidra.program.model.data.LongDoubleDataType):
        return "float"

    return "unsigned"

def dump_data_types(stream, filter=None):
    db = {}
    manager = currentProgram.getDataTypeManager()
    for dt in manager.getAllDataTypes():
        name = dt.getName()
        ty = Type()
        ty.size = dt.getLength()
        if ty.size < 0:
            continue

        if filter != None and not filter(name):
            continue

        if isinstance(dt, ghidra.program.database.data.TypedefDB):
            dt = dt.getBaseDataType()
            if name != dt.getName():
                ty.typedef = dt.getName()
                ty.flags.append("typedef")

        ty.flags.append(get_type_attribute(dt))
        
        if isinstance(dt, ghidra.program.database.data.PointerDB):
            pdt = dt.getDataType()
            ty.ref = ""
            if pdt != None:
                ty.ref = pdt.getName()
        elif isinstance(dt, ghidra.program.database.data.ArrayDB):
            ty.element = dt.getDataType().getName()
            ty.length = dt.getNumElements()
        elif isinstance(dt, ghidra.program.database.data.StructureDB) or isinstance(dt, ghidra.program.database.data.UnionDB):
            ty.members = []
            for comp in dt.getComponents():
                cdt = comp.getDataType()
                typename = cdt.getName()
                comment = comp.getComment()
                desc = ""
                if comment != None:
                    desc = str(comment)

                if isinstance(cdt, ghidra.program.model.data.StringDataType):
                    length = comp.getLength()
                    typename = "char[%d]" % (length)
                    db[typename] = { "element": "char", "flags": ["array"], "length": length, "size": length }
                ty.members.append([comp.getFieldName(), typename, desc])

        db[name] = ty

    json.dump(db, stream, sort_keys=True, cls=Encoder, indent=4)

dump_data_types(sys.stdout)
