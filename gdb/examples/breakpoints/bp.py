"""

Test of setting breakpoints in python and catching it
Inside the breakpoint class, it is not allowed to modify/delete/add breakpoints

"""
import gdb

class FunctionBreakpoint(gdb.Breakpoint):
    def __init__(self, name):
        self.seen = False
        self.name = name
        self.count = 0
        gdb.Breakpoint.__init__(self, name)

    def stop(self):
        frame = gdb.selected_frame()
        self.count += 1
        # only print function names we haven't seen before
        if not self.seen:
            self.seen = True
            print(frame.name())

def stats(funcs):
    for f in funcs:
        print(f.name, f.count)

funcs = []
for i in range(1000):
    funcs.append(FunctionBreakpoint(f"f{i}"))

