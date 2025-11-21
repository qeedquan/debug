# Generate C function stubs
#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

import ghidra.app.decompiler as decompiler
import sys

def gen_c_data_stubs(stream):
    stream.write("#include <stdio.h>\n")
    stream.write("#include <stdint.h>\n")
    stream.write("#include <wchar.h>\n")
    stream.write("#include <math.h>\n")
    stream.write("#define __stdcall __attribute__((stdcall))\n")
    stream.write("#define __fastcall __attribute__((fastcall))\n")
    stream.write("#define __cdecl __attribute__((cdecl))\n")
    stream.write("typedef unsigned char uchar;\n")
    stream.write("typedef unsigned short ushort;\n")
    stream.write("typedef unsigned int uint;\n")
    stream.write("typedef unsigned long long ullong;\n")
    stream.write("typedef int undefined;\n")
    stream.write("typedef uchar undefined8;\n")
    stream.write("\n")

def gen_c_function_stubs(stream):
    decomp = decompiler.DecompInterface()
    decomp.openProgram(currentProgram)
    funcmgr = currentProgram.getFunctionManager()
    funcs = funcmgr.getFunctionsNoStubs(True)

    seen = {}
    funcid = 0
    for func in funcs:
        try:
            if func.isExternal():
                continue

            prototype = func.getPrototypeString(True, True)
            if "FID_conflict" in prototype or "@" in prototype or "__cdecl" in prototype or "round" in prototype or "allocate" in prototype:
               continue
            if prototype in seen:
                func.setName(func.getName() + "__" + str(funcid), SourceType.DEFAULT)
                funcid += 1
                prototype = func.getPrototypeString(True, True)
                continue

            seen[prototype] = True

            stream.write("// %s\n" % (str(func.getEntryPoint())))
            stream.write("%s {}\n" % prototype)
        except:
            stream.write("// %s: decompile failed" % (func.getName()))
        stream.write("\n\n")

gen_c_data_stubs(sys.stdout)
gen_c_function_stubs(sys.stdout)
