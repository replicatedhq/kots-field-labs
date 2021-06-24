#!/usr/bin/env python

import sys,os
import curses

class menu:
    current_menu = []
    menu_stack = []
    deep = []
    select_ks = ["Select KS file:"]
    menu0_0=["create", "chroot", "covert"]
    menu1_0=["loop", "raw", "fs", "livecd", "liveusb"]
    menu1_1=[" "]
    menu1_2=[" "]
    menu2_1=["select image file"]
    menu2_2=["select image file"]
    menu2_0=select_ks
    menu3_1=["-h", "--help", "-s", "--saveto"]
    menu3_2=["loop", "raw", "fs", "livecd", "liveusb"]
    menu3_0=["-h", "-c", "-o", "-A"]
    menu4_0=[]
    menu4_1=[]
    menu4_2=["-h", "-S", "--shell"]
    spec=["Command", "Image Type", "SK file or image file", "Options", "Options"]

    def __init__(self):
        self.menu_stack.append(self.menu0_0)
        self.current_menu=self.menu0_0

    def get_current_menu(self):
        return self.current_menu

    def menu_push(self, item):
        print "menu_push"
        self.menu_stack.append(item)
        self.current_menu = item

    def menu_pop(self):
        self.menu_stack.pop()
        if len(self.menu_stack) != 0 and isinstance(self.menu_stack[len(self.menu_stack) - 1],list):
            self.current_menu = self.menu_stack[len(self.menu_stack) - 1]
        else:
            self.current_menu = self.menu_stack


def mymenu(stdscr):
    k=0
    c_item = 0
    l_item = 0
    cmdline=["mic"]
    menu_item=[]
    c_menu = menu()
    c_menu_len = 0
    stdscr.clear()
    stdscr.refresh()
    cmd_num = 0
    height,width = stdscr.getmaxyx()

    while k != ord('q'):

        c_menu_len = len(c_menu.get_current_menu())
        if k == curses.KEY_DOWN:
            if c_item < c_menu_len - 1:
                c_item += 1
        elif k == curses.KEY_UP:
            if c_item >= 1:
                c_item -= 1
        elif k == curses.KEY_RIGHT:
            l_item += 1
            if hasattr(c_menu, "menu%d_%x" % (l_item, cmd_num)):
                menu_item=getattr(c_menu, "menu%d_%d" % (l_item, cmd_num))

            cmdline.append(c_menu.get_current_menu()[c_item])

            c_menu.menu_push(menu_item)
            c_item = 0

        elif k == curses.KEY_LEFT:
            c_menu.menu_pop()
            if l_item >= 1:
                l_item -= 1
            cmdline.pop()
            c_item = 0

        stdscr.clear()

        subwin = stdscr.subwin(0, width, 0, 0)
        subwin.box()
        subwin.move(0, 0)
        subwin.addstr(getattr(c_menu, "spec")[l_item])
        cliwin = stdscr.subwin(0, width, height-5, 0)
        cliwin.box()
        cliwin.move(0, 0)
        cliwin.addstr("Mic cmd:")

        print c_menu.get_current_menu()
        for i in range(len(c_menu.get_current_menu())):
            stdscr.move(i+2,5)
            stdscr.addstr("%s" % c_menu.get_current_menu()[i])

        stdscr.move(height-3, 5)
        for i in range(len(cmdline)):
            stdscr.addstr("%s " % cmdline[i])
        stdscr.addstr("%s " % c_menu.get_current_menu()[c_item])

        stdscr.move(c_item + 2, 5)
        stdscr.refresh()
        if l_item == 0:
            cmd_num = c_item
        k = stdscr.getch()



def main():
    curses.wrapper(mymenu)




if __name__ == "__main__":
    main()