#!/usr/bin/env python

# Copyright 2018 Arm Limited.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import _curses
import argparse
import curses
import importlib
import logging
import os
import sys

# This script is actually within our package, so add the package to the python path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from config_system import general
from config_system import log_handlers

logger = logging.getLogger(__name__)

mainwindow_help_text = "Use arrow keys to navigate the menu. <Enter> selects submenus. Pressing <Y> enables option, "\
        "<N> disables. Press <Esc><Esc> to exit, <?> for Help, </> for Search, <r> to Reset option to default value"
# Character '@' will be replaced with ' ' and set appropriate color
legend_help_text =  "Legend: [*] - enabled, [ ] - disabled, [@] - set by user"

save_configuration_text = "Do you wish to save your new configuration?"

title_bar = '' # will be set in main

attr = {}
menustack = []

def init_attr():
    curses.init_pair(1, curses.COLOR_WHITE, curses.COLOR_BLUE)
    curses.init_pair(2, curses.COLOR_BLACK, curses.COLOR_WHITE)
    curses.init_pair(3, curses.COLOR_WHITE, curses.COLOR_WHITE)
    curses.init_pair(4, curses.COLOR_BLUE, curses.COLOR_WHITE)
    curses.init_pair(5, curses.COLOR_GREEN, curses.COLOR_WHITE)
    # Color for set by user
    curses.init_pair(6, curses.COLOR_WHITE, curses.COLOR_YELLOW)

    attr['bg'] = curses.color_pair(1)
    attr['window'] = curses.color_pair(2)
    attr['shadow'] = curses.color_pair(2) | curses.A_REVERSE
    attr['lit'] = curses.color_pair(3) | curses.A_BOLD
    attr['title'] = curses.color_pair(4) | curses.A_BOLD
    attr['highlight'] = curses.color_pair(1) | curses.A_BOLD
    attr['scroll'] = curses.color_pair(5) | curses.A_BOLD
    attr['option_set_by_user'] = curses.color_pair(6) | curses.A_BOLD



class MenuBar(object):
    def __init__(self, options):
        self.options = options
        self.selection = 0
    def left(self):
        if self.selection > 0:
            self.selection -= 1
    def right(self):
        if self.selection < len(self.options)-1:
            self.selection += 1
    def get_selection(self):
        return self.options[self.selection]

def lit_border(window, h, w, y=0, x=0, raised=True, bar=None):
    if raised:
        window.attrset(attr['lit'])
    else:
        window.attrset(attr['window'])
    window.hline(y, x+1, curses.ACS_HLINE, w-2)
    window.vline(y+1, x, curses.ACS_VLINE, h-2)
    window.addch(y, x, curses.ACS_ULCORNER)
    window.addch(y+h-1, x, curses.ACS_LLCORNER)
    if bar:
        window.hline(y+h-3, x+1, curses.ACS_HLINE, w-2)
        window.addch(y+h-3, x, curses.ACS_LTEE)

    if raised:
        window.attrset(attr['window'])
    else:
        window.attrset(attr['lit'])
    window.hline(y+h-1, x+1, curses.ACS_HLINE, w-2)
    window.vline(y+1, x+w-1, curses.ACS_VLINE, h-2)
    window.addch(y, x+w-1, curses.ACS_URCORNER)
    try:
        # The lower right corner can cause an exception, but the character is
        # written anyway
        window.addch(y+h-1, x+w-1, curses.ACS_LRCORNER)
    except _curses.error as e:
        pass

    if bar:
        window.addch(y+h-3, x+w-1, curses.ACS_RTEE)
        bar_width = 0
        for op in bar.options:
            bar_width += len(op) + 5
        x = ((x+w)-(bar_width))//2
        if x < 0:
            x = 0
        for i in range(0,len(bar.options)):
            a = attr['window']
            if bar.selection == i:
                a = attr['highlight']
                bar.selection_pos = (y+h-2, x+1)
            window.addstr(y+h-2, x, "< %s >" % bar.options[i], a)
            x += len(bar.options[i])+5
            if x >= w:
                break

    window.attrset(attr['window'])

def window_border(window, title, menu_bar):
    (h, w) = window.getmaxyx()

    lit_border(window, h, w, bar=menu_bar)

    if title == None:
        return

    title = " "+title+" "

    x = (w-len(title))//2
    if x < 0:
        x = 0
    window.addstr(0, x, title, attr['title'])

def wrap_text(window, text, y, x, w, max_y=None, y_offset=0):
    ypos = y - y_offset
    while text != "":
        next_text = ""
        line_break = text.find("\n")
        if line_break != -1 or len(text) > w:
            if line_break > w or line_break == -1:
                b = text.rfind(" ", 0, w)
                if b < 1:
                    b = w
            else:
                b = line_break
            next_text = text[b+1:]
            text = text[:b]

        if window and ypos >= y and (max_y == None or ypos < max_y):
            window.addstr(ypos, x, text)

        ypos += 1

        text = next_text
    return ypos

def prepare_wrap_text(text, width):
    wrapped_text = ""
    while text != "":
        next_text = ""
        line_break = text.find("\n")
        if line_break != -1 or len(text) > width:
            if line_break > width or line_break == -1:
                b = text.rfind(" ", 0, width)
                if b < 1:
                    b = width
            else:
                b = line_break
            next_text = text[b+1:]
            text = text[:b]

        wrapped_text += text
        wrapped_text += '\n'

        text = next_text
    if wrapped_text[-1] == '\n':
        wrapped_text = wrapped_text[:-1]
    return wrapped_text

def draw_text(window, text, y, x):
    for text_line in text.splitlines():
        window.addstr(y, x, text_line)
        y += 1
    return y

def draw_background(stdscr):
    (height, width) = stdscr.getmaxyx()

    stdscr.bkgd(' ', attr['bg'])
    stdscr.erase()
    stdscr.addstr(0, 1, title_bar, curses.A_BOLD)
    stdscr.hline(1, 1, curses.ACS_HLINE, width-2)

def fit_window(stdscr, win_h, win_w):
    (height, width) = stdscr.getmaxyx()
    if win_h == None:
        (win_h, win_w) = (height-4, width-5)
    else:
        if win_h > height-4:
            win_h = height-4
        if win_w > width-5:
            win_w = width-5

    y = int((height-win_h)/2)
    x = int((width-win_w)/2)

    return (win_h, win_w, y, x)

def draw_window(stdscr, window, title, menu_bar, win_h = None, win_w = None):
    (height, width) = stdscr.getmaxyx()
    (win_h, win_w, y, x) = fit_window(stdscr, win_h, win_w)

    window.resize(win_h, win_w)
    window.mvwin(y, x)
    window.bkgd(' ', attr['window'])
    window.erase()

    # Drop shadow
    stdscr.attrset(attr['shadow'])
    stdscr.hline(win_h+y, x+1, ' ', win_w)
    stdscr.vline(y+1, win_w+x, ' ', win_h)
    stdscr.vline(y+1, win_w+x+1, ' ', win_h)

    window_border(window, title, menu_bar)

    return (win_h, win_w)

def draw_legend(window, y, x, width):
    legend = prepare_wrap_text(legend_help_text, width)

    legend_draw_start = y
    y = draw_text(window, legend, y, x)

    for text_line in legend.splitlines():
        position = text_line.find("@")
        if position != -1:
            window.addch(legend_draw_start, x + position, ' ', attr['option_set_by_user']) # override char
        legend_draw_start += 1
    return y

def draw_main_menu(stdscr, window, menu, menu_bar):
    draw_background(stdscr)
    (win_h, win_w) = draw_window(stdscr, window, menu.title, menu_bar)
    x = 3

    y = wrap_text(window, mainwindow_help_text, 1, x, win_w-6)
    y = draw_legend(window, y, x, win_w-6)

    menu_height = win_h - y - 3

    lit_border(window, menu_height, win_w - 4, y, 2, raised=False)

    y += 1
    menu_height -= 2
    menu_bottom = menu_height + y
    menu_top = y
    x = 7

    if menu.selection < menu.top:
        menu.top = menu.selection

    # Check if the menu could be scrolled up
    can_scroll_up = False
    for i in range(0, menu.top):
        if menu[i].can_enable() and menu[i].is_visible():
            can_scroll_up = True
            break

    # Compute which menu items should be visible and scrolling
    menu_items = []
    can_scroll_down = True
    for i in range(menu.top, len(menu.items)):
        if menu[i].can_enable() and menu[i].is_visible():
            menu_items.append(i)
        if len(menu_items) >= menu_height:
            if menu.selection > i:
                menu_items = menu_items[1:]
                if len(menu_items) > 0:
                    menu.top = menu_items[0]
                else:
                    # Screen is too small to show any items
                    raise _curses.error("Too small")
                can_scroll_up = True
            else:
                for j in range(i+1, len(menu.items)):
                    if menu[j].can_enable() and menu[i].is_visible():
                        break
                else:
                    can_scroll_down = False
                break
    else:
        can_scroll_down = False

    cursor_y = 0

    max_width = win_w - x - 3

    if can_scroll_up:
        window.addstr(menu_top-1, min(7, win_w-3), " ^ ", attr['scroll'])
    if can_scroll_down:
        window.addstr(menu_bottom, min(7, win_w-3), " v ", attr['scroll'])

    for menu_pos in menu_items:
        menu_option = menu[menu_pos]

        is_selected = False
        if menu.selection == menu_pos:
            cursor_y = y
            is_selected = True

        tmp_x = x
        remaining_width = max_width
        for part in menu[menu_pos].get_styled_text(is_selected, max_width):
            if len(part.text) > 0 and remaining_width > 0:
                window.addstr(y, tmp_x, part.text[:remaining_width], attr[part.style])
                tmp_x += len(part.text)
                remaining_width -= len(part.text)

        y += 1

    window.move(cursor_y, x+1)

    return menu_height

def draw_prompt(stdscr, window, menu_bar, prompt, input_box=None, title=None,
        cursor_pos=0, scroll_pos=0):
    draw_background(stdscr)

    (height, width) = stdscr.getmaxyx()

    x_padding = 3
    text_width = len(prompt)

    (win_h, win_w, y, x) = fit_window(stdscr, 1, text_width+x_padding*2)

    text_width = min(win_w-x_padding*2, text_width)
    win_w = text_width + x_padding*2

    # Calculate the number of lines needed
    text_height = wrap_text(None, prompt, 4, x_padding, text_width)

    if input_box != None:
        text_height += 3
        text_box_width = min(max(70, len(input_box)+3), width-8)
        if text_box_width < len(input_box)+3:
            trim_amt = len(input_box)+3-text_box_width
            if trim_amt > cursor_pos-10:
                trim_amt = max(cursor_pos-10, 0)
                input_box = input_box[trim_amt:trim_amt+text_box_width-2]
            else:
                input_box = input_box[trim_amt:]
            cursor_pos -= trim_amt
        win_w = max(win_w, text_box_width+4)

    (win_h, win_w) = draw_window(stdscr, window, title, menu_bar,
            text_height, win_w)

    wrap_text(window, prompt, 1, x_padding, text_width, max_y=win_h-3,
            y_offset = scroll_pos)

    if input_box != None:
        lit_border(window, 3, text_box_width, win_h-6, 2)
        window.addstr(win_h-5, 3, input_box)
        window.move(win_h-5, 3+cursor_pos)

    # Return the amount of scroll possible and the size of a page
    return (text_height - win_h, win_h-3)

def prompt(stdscr, window, text, options=["OK"]):
    menu_bar = MenuBar(options)
    scroll_pos = 0

    while True:
        (max_scroll, page_size) = draw_prompt(stdscr, window, menu_bar, text,
                scroll_pos=scroll_pos)
        window.move(*menu_bar.selection_pos)

        stdscr.noutrefresh()
        window.noutrefresh()
        curses.doupdate()

        c = stdscr.getch()

        if c == curses.KEY_RIGHT:
            menu_bar.right()
        elif c == curses.KEY_LEFT:
            menu_bar.left()
        elif c == curses.KEY_UP:
            scroll_pos -= 1
        elif c == curses.KEY_DOWN:
            scroll_pos += 1
        elif c == curses.KEY_NPAGE:
            scroll_pos += page_size
        elif c == curses.KEY_PPAGE:
            scroll_pos -= page_size
        elif c == curses.KEY_HOME:
            scroll_pos = 0
        elif c == curses.KEY_END:
            scroll_pos = max_scroll
        elif c == 10:
            cmd = menu_bar.get_selection()
            return cmd

        if scroll_pos > max_scroll:
            scroll_pos = max_scroll
        if scroll_pos < 0:
            scroll_pos = 0

def inputbox(stdscr, window, value="", title="", prompt="Please enter a value"):
    menu_bar = MenuBar(["Ok"])

    cursor_pos = len(value)

    while True:
        draw_prompt(stdscr, window, menu_bar, prompt,
                input_box = value, title = title,
                cursor_pos = cursor_pos)

        stdscr.noutrefresh()
        window.noutrefresh()
        curses.doupdate()

        c = stdscr.getch()

        if c == 10:
            return (True,  value)
        elif c == 27:
            return (False, None)
        elif c == curses.KEY_BACKSPACE:
            if cursor_pos > 0:
                cursor_pos -= 1
                value = value[:cursor_pos]+value[cursor_pos+1:]
        elif c == curses.KEY_DC:
            value = value[:cursor_pos]+value[cursor_pos+1:]
        elif c < 256 and c >= 32:
            value = value[:cursor_pos]+chr(c)+value[cursor_pos:]
            cursor_pos += 1
        elif c == curses.KEY_LEFT:
            if cursor_pos > 0:
                cursor_pos -= 1
        elif c == curses.KEY_RIGHT:
            if cursor_pos < len(value):
                cursor_pos += 1
        elif c == curses.KEY_HOME:
            cursor_pos = 0
        elif c == curses.KEY_END:
            cursor_pos = len(value)

def item_inputbox(stdscr, window, menu_item):
    (success, value) = inputbox(stdscr, window,
            value=menu_item.get_value(), title=menu_item.get_title())
    if success:
        menu_item.set_value(value)

def get_menu_location(value):
    for i in general.menu_data:
        for k in general.menu_data[i]:
            if k.value == value:
                parent = [general.get_menu_title(i)]
                parent += get_menu_location(i)
                return parent
    return []

def search(stdscr, window):
    (success, string) = inputbox(stdscr, window,
            title="Search", prompt="Enter substring to search for")
    if not success:
        return
    string = string.lower()
    results = ""
    for i in general.get_config_list():
        config = general.get_config(i)
        if (string in i.lower() or
                string in (config.get('title') or "").lower()
                or string in (config.get('help') or "").lower()):
            results += "Symbol: %s [=%s]\n" % (i,config['value'])
            results += "Type  : %s\n" % (config['datatype'],)
            if 'title' in config:
                results += "Prompt: %s\n" % (config['title'],)
            menu_stack = get_menu_location(i)
            if len(menu_stack) > 0:
                results += "  Location:\n"
            indent = 2
            while len(menu_stack) > 0:
                m = menu_stack.pop()
                indent += 2
                results += ' ' * indent
                results += "-> %s\n" % (m,)
            results += "\n\n"
    if results == "":
        results = "No matches found"
    prompt(stdscr, window, results)

def main(stdscr):
    (height, width) = stdscr.getmaxyx()

    init_attr()

    window = curses.newwin(2, 2, 2, 2)

    menu_bar = MenuBar(["Select","Exit","Help"])

    while len(menustack) > 0:
        menu = menustack[-1]
        global title_bar
        title_bar = menu.title_bar
        try:
            menu_height = draw_main_menu(stdscr, window, menu, menu_bar)
        except _curses.error:
            (height, width) = stdscr.getmaxyx()
            stdscr.bkgd(' ', attr['bg'])
            stdscr.erase()
            try:
                stdscr.attrset(attr['title'])
                wrap_text(stdscr, "Terminal too small?", 0, 0, width)
            except _curses.error:
                pass

        stdscr.noutrefresh()
        window.noutrefresh()
        curses.doupdate()

        c = stdscr.getch()

        selection = menu.get_selection()

        def move_up():
            global sel
            sel = menu.selection
            while sel > 0:
                sel -= 1
                if menu[sel].can_enable() and menu[sel].is_visible():
                    menu.selection = sel
                    return
        def move_down():
            global sel
            sel = menu.selection
            while sel < len(menu.items)-1:
                sel += 1
                if menu[sel].can_enable() and menu[sel].is_visible():
                    menu.selection = sel
                    return

        if c == 27:
            menustack.pop()
        elif c == curses.KEY_DOWN:
            move_down()
        elif c == curses.KEY_UP:
            move_up()
        elif c == curses.KEY_NPAGE:
            for i in range(menu_height):
                move_down()
        elif c == curses.KEY_PPAGE:
            for i in range(menu_height):
                move_up()
        elif c == curses.KEY_RIGHT:
            menu_bar.right()
        elif c == curses.KEY_LEFT:
            menu_bar.left()
        elif c == 10:
            cmd = menu_bar.get_selection()
            if cmd == "Select":
                if selection.is_menu():
                    menustack.append(selection.get_menu())
                elif selection.needs_inputbox():
                    item_inputbox(stdscr, window, selection)
                elif selection.select():
                    menustack.pop()
            elif cmd == "Exit":
                menustack.pop()
            elif cmd == "Help":
                prompt(stdscr, window, selection.get_help())
            menu_bar.selection = 0
        elif c == ord('y'):
            selection.set_bool('y')
        elif c == ord('n'):
            selection.set_bool('n')
        elif c == ord(' '):
            selection.toggle()
        elif  c == ord('r'):
            selection.reset()
        elif c == curses.KEY_REFRESH or c == 12: # Ctrl-L
            stdscr.clear()
        elif c == ord('?'):
            prompt(stdscr, window, selection.get_help())
        elif c == ord('/'):
            search(stdscr, window)

    if prompt(stdscr, window, save_configuration_text, ["Yes","No"]) == "Yes":
        return True
    return False

if __name__ == "__main__":
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.WARNING)

    formatter = logging.Formatter('%(levelname)s: %(message)s')

    # The eventual destination is stdout with the above formatting
    errHandler = logging.StreamHandler(sys.stderr)
    errHandler.setFormatter(formatter)

    # Setup a buffer to store messages
    msgBuffer = log_handlers.InfBufferHandler(8192, target=errHandler)
    root_logger.addHandler(msgBuffer)

    # Also count each type of message
    counter = log_handlers.ErrorCounterHandler()
    root_logger.addHandler(counter)

    parser = argparse.ArgumentParser()
    parser.add_argument('config', help='Path to the input configuration file (*.config)')
    parser.add_argument('-o', '--output',
                        help='Path to the output file')
    parser.add_argument('-d', '--database', default="Mconfig",
                        help='Path to the configuration database (Mconfig)')
    parser.add_argument('--debug', action='store_true', dest='debug',
                        help='Enable debug logging')
    parser.add_argument('-p', '--plugin', action='append',
                        help='Post configuration plugin to execute',
                        default=[])
    parser.add_argument('--ignore-missing', dest="ignore_missing", action='store_true', default=False,
                        help="Ignore missing database files included with 'source'")
    parser.add_argument('args', nargs="*")
    args = parser.parse_args()

    if args.output is None:
        args.output = args.config
    if args.debug:
        root_logger.setLevel(logging.DEBUG) # Update root logger

    general.read_config(args.database, args.config, args.ignore_missing)

    menustack.append(general.get_root_menu())

    if curses.wrapper(main):
        general.enforce_dependent_values()
        for plugin in args.plugin:
            path, name = os.path.split(plugin)
            if path.startswith('/'):
                sys.path.insert(0, path)
            else:
                sys.path.insert(0, os.path.join(os.getcwd(), path))
            sys.path.append(os.path.dirname(sys.argv[0]))
            try:
                mod = importlib.import_module(name)
                mod.plugin_exec()
            except ImportError as err:
                logger.error(err)
            except Exception as err:
                logger.warning("Problem encountered in %s plugin: %s" % (name, repr(err)))
                import traceback
                traceback.print_tb(sys.exc_info()[2])

        general.write_config(args.output)

    # Flush all log messages on exit
    msgBuffer.close()

    issues = counter.errors() + counter.criticals()
    sys.exit(0 if issues == 0 else 1)
