# Generate stubs for external functions
#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

import ghidra.app.decompiler as decompiler
import sys

def is_valid_namespace(namespace):
    value = str(namespace)
    if value.find("(") >= 0 or value.find(".") >= 0:
        return False

    if value == "__FrameHandler3":
        return False

    return True

def classify_type(kind):
    desc = kind.getDescription()
    if desc.startswith("Signed") or desc.startswith("Unsigned") or desc.startswith("pointer"):
        return 'i'
    return 'u'

def gen_stubs(stream):
    listing = currentProgram.getListing()
    namespaces = []
    func = getFirstFunction()
    while func is not None:
        namespace = func.getParentNamespace()
        if is_valid_namespace(namespace) and namespace not in namespaces:
            namespaces.append(namespace)

        func = getFunctionAfter(func)

    for namespace in namespaces:
        if not namespace.isGlobal():
            stream.write("namespace %s {\n\n" % str(namespace))
    
        func = getFirstFunction()
        while func is not None:
            namespacefn = func.getParentNamespace()
            proto = func.getPrototypeString(True, False)
            protostr = str(proto)
            return_type = func.getReturnType()
            if namespacefn == namespace and not protostr.startswith("undefined"):
                stream.write(proto)
                stream.write("{\n")
                if classify_type(return_type) == 'i':
                    stream.write("\treturn 0;\n")
                stream.write("}\n\n")

            func = getFunctionAfter(func)

        if not namespace.isGlobal():
            stream.write("} // END OF NAMESPACE %s\n\n" % str(namespace))

gen_stubs(sys.stdout)
