# Find duplicated functions
#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

class Func:
    def __init__(self):
        self.func = None
        self.body = []
        self.hash = ""

def getfuncs():
    listing = currentProgram.getListing()
    result = []
    func = getFirstFunction()
    while func is not None:
        f = Func()
        f.func = func
        f.body = list(listing.getCodeUnits(func.getBody(), True))
        for op in f.body:
            f.hash += op.toString().split(" ")[0]
        result.append(f)
        func = getFunctionAfter(func)
    return result

def getdiffs(funcs):
    dups = {}
    for f in funcs:
        key = f.hash
        if key not in dups:
            dups[key] = []
        dups[key].append(f.func)

    for key in dups:
        if len(dups[key]) > 1:
            print(dups[key])

getdiffs(getfuncs())
