# Rename struct names
#@author 
#@category CUSTOM
#@keybinding 
#@menupath 
#@toolbar 

def rename_struct(old, new):
    dtm = currentProgram.getDataTypeManager()
    if dtm == None:
        return
    
    oldStruct = dtm.getDataType(old)
    if oldStruct == None:
        print("Structure '%s' not found" % old)
        return

    transactionID = currentProgram.startTransaction("Rename Structure")
    try:
        oldStruct.setName(new)
    except Exception as exception:
        print("Error renaming struct: ", exception)
    finally:
        currentProgram.endTransaction(transactionID, True)

def remove_struct(name):
    dtm = currentProgram.getDataTypeManager()
    if dtm == None:
        return

    dt = dtm.getDataType(name)
    if dt != None:
        dtm.remove(dt, monitor)

# example usage
# rename_struct("/struct.h/OldStruct", "NewStruct")
# remove_struct("/struct.h/StructName")

