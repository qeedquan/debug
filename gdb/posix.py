def list_all_signal_handlers():
    gdb.execute("set $p = (struct sigaction *) malloc(sizeof (struct sigaction))")
    for i in range(1, 65):
        gdb.execute("call sigaction(%d, 0, $p)" % (i))
        print(("Signal %d %s") % (i, str(gdb.parse_and_eval("$p->__sigaction_handler.sa_handler"))))
    gdb.execute("call free($p)")

