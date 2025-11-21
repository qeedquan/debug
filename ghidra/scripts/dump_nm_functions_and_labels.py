# Dump function/labels in nm format
#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

import sys

def is_data_label(name):
    if "LAB_" in name:
        return False
    if "switchD" in name:
        return False
    if "switchdataD_" in name:
        return False
    if "caseD_" in name:
        return False
    if "::" in name or "?" in name or "@" in name:
        return False
    if "FuncInfo" in name:
        return False
    if "Unwind" in name:
        return False
    if "RTTI" in name:
        return False
    if "vftable" in name:
        return False
    return True

def dump_functions_and_labels(stream=sys.stdout, base=0, reloc=0, fmt=""):
    program = getCurrentProgram()
    sym_table = program.getSymbolTable()

    func = getFirstFunction()
    num_funcs = 0
   
    image_base = int(str(currentProgram.getImageBase()), 16)
    
    off = image_base
    entry = int(str(func.getEntryPoint()), 16)
    if entry >= off:
        off = entry - off

    while func is not None:
        body = func.getBody()
        entry = func.getEntryPoint()
        start = int(str(entry), 16) - base + reloc
        end = int(str(body.getMaxAddress()), 16) - base + reloc
        size = end - start + 1
        name = func.getName()
        if func.isThunk():
            name += "_Thunk_%X" % (entry.getOffset())

        if fmt == "nm":
            stream.write("{:#x} T {}\n".format(start, name))
        else:
            stream.write("{:#x} {:#x} {:#x} {:#x} r--p {} T\n".format(start, end, size, off, name))

        func = getFunctionAfter(func)
        num_funcs += 1
        off += size

    num_labels = 0
    for sym in sym_table.getAllSymbols(True):
        kind = str(sym.getSymbolType())
        name = str(sym.getName(False))
        addr = sym.getAddress()
        start = addr.getOffset()
        size = addr.getSize()
        end = start + size - 1
        off = start

        if kind != "Label":
            continue
        if not is_data_label(name):
            continue

        if fmt == "nm":
            stream.write("{:#x} D {}\n".format(start, name))
        else:
            stream.write("{:#x} {:#x} {:#x} {:#x} r--p {} D\n".format(start, end, size, off, name))
        
        num_labels += 1

    stream.write("Number of functions: {}\n".format(num_funcs))
    stream.write("Number of labels: {}\n".format(num_labels))
    stream.write("Image Base: %#x\n" % (image_base))
    stream.write("Base: %#x\n" % (base))
    stream.write("Relocation: %#x\n" % (reloc))

file = askFile("Save to file", "Output")
stream = open(file.absolutePath, "w")
dump_functions_and_labels(stream=stream)
stream.close()
