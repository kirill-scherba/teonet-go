// Copyright 2019 Kirill Scherba <kirill@scherba.ru> as Teokeys Author.
// All rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package teokeys

import (
	"fmt"
	"strings"
	"time"
)

// CLI hotkeys processing module
// Created 2019-08-13 by Kirill Scherba <kirill@scherba.ru>

// Hotkey data for process hokey
type Hotkey struct {
	hkey  []int  // list of hotkeys
	usage string // this key usage (help) string
	f     func() // function exequted when this key pressed
}

// HotkeyMenu structure to hold hotkey menu definitions and methods
type HotkeyMenu struct {
	usageTitle string
	pressed    string
	quitF      bool
	menu       []Hotkey
}

// CreateMenu create hotkey menu
func CreateMenu(usageTitle, pressed string) (hk *HotkeyMenu) {
	hk = &HotkeyMenu{usageTitle: usageTitle, pressed: pressed}
	return
}

// Add add record to hotkey menu
func (hk *HotkeyMenu) Add(hkey []int, usage string, f func()) {
	hk.menu = append(hk.menu, Hotkey{hkey: hkey, usage: usage, f: f})
}

var line = strings.Repeat("-", 68) + "\n"

// Usage show hotkey menu
func (hk *HotkeyMenu) Usage() {
	if hk.usageTitle != "" {
		fmt.Print(hk.usageTitle + "\n" + line)
	} else {
		fmt.Print(line)
	}
	for _, item := range hk.menu {
		fmt.Printf(Color(ANSIGreen, string(item.hkey[0])) + " - " + item.usage + "\n")
	}
	fmt.Print(line)
}

// Getch wait and return hotkey
func (hk *HotkeyMenu) Getch() (ch int) {
	for !hk.quitF {
		ch = GetchNb()
		if ch != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return
}

// Process hotkeys
func (hk *HotkeyMenu) Process(ch int) {

	// Check hotkey present in array of hotkeys
	checkKeys := func(ch int, hkey []int) bool {
		for _, k := range hkey {
			if k == ch {
				return true
			}
		}
		return false
	}

	// Find record suitable for elected hotkey
	for _, item := range hk.menu {
		if checkKeys(ch, item.hkey) {
			if item.f != nil {
				item.f()
			}
			break
		}
	}
}

// Run get char and process it
func (hk *HotkeyMenu) Run() {
	for {
		ch := hk.Getch()
		if ch == 0 {
			break
		}
		if hk.pressed != "" {
			fmt.Printf("\b"+hk.pressed, ch)
		}
		hk.Process(ch)
	}
}

// Process quit from Getch(). And the Getch() will reurn 0
func (hk *HotkeyMenu) Quit() {
	hk.quitF = true
}