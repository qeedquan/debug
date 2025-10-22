#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

import ghidra.app.decompiler as decompiler
import sys

def decompile_all_functions(stream):
    decomp = decompiler.DecompInterface()
    decomp.openProgram(currentProgram)
    funcmgr = currentProgram.getFunctionManager()
    funcs = funcmgr.getFunctions(True)
    for func in funcs:
        try:
            res = decomp.decompileFunction(func, 10, None)
            dec = res.getDecompiledFunction()
            if dec == None:
                continue

            stream.write(dec.getC())
            stream.write("\n")
        except:
            stream.write("/* Failed to decompile */\n")

file = askFile("Save to file", "Output")
stream = open(file.absolutePath, "w")
decompile_all_functions(stream)
stream.close()
