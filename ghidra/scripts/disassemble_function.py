#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

from binascii import hexlify

def disassemble_function(name):
    listing = currentProgram.getListing()
    funcs = getGlobalFunctions(name)
    if len(funcs) == 0:
        return
    func = funcs[0]
    addrset = func.getBody()
    codeunit = listing.getCodeUnits(addrset, True)
    num_insts = 0
    for code in codeunit:
        print("0x{} : {:24} {}".format(code.getAddress(), hexlify(code.getBytes()), code.toString()))
        num_insts += 1
    print("Instructions Count: {}".format(num_insts))

