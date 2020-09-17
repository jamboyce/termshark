// Copyright 2019-2020 Graham Clark. All rights reserved.  Use of this source
// code is governed by the MIT license that can be found in the LICENSE
// file.

// Package ui contains user-interface functions and helpers for termshark.
package ui

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	"github.com/gcla/gowid/vim"
	"github.com/gcla/termshark/v2"
	"github.com/gcla/termshark/v2/theme"
	"github.com/gcla/termshark/v2/widgets/mapkeys"
	"github.com/gcla/termshark/v2/widgets/minibuffer"
	"github.com/gdamore/tcell/terminfo"
	"github.com/gdamore/tcell/terminfo/dynamic"
	"github.com/rakyll/statik/fs"
	"github.com/shibukawa/configdir"

	_ "github.com/gcla/termshark/v2/assets/statik"
)

//======================================================================

var notEnoughArgumentsErr = fmt.Errorf("Not enough arguments provided")
var invalidSetCommandErr = fmt.Errorf("Invalid set command")
var invalidReadCommandErr = fmt.Errorf("Invalid read command")
var invalidRecentsCommandErr = fmt.Errorf("Invalid recents command")
var invalidMapCommandErr = fmt.Errorf("Invalid map command")
var invalidFilterCommandErr = fmt.Errorf("Invalid filter command")
var invalidThemeCommandErr = fmt.Errorf("Invalid theme command")

type minibufferFn func(gowid.IApp, ...string) error

func (m minibufferFn) Run(app gowid.IApp, args ...string) error {
	return m(app, args...)
}

func (m minibufferFn) OfferCompletion() bool {
	return true
}

func (m minibufferFn) Arguments([]string) []minibuffer.IArg {
	return nil
}

type quietMinibufferFn func(gowid.IApp, ...string) error

func (m quietMinibufferFn) Run(app gowid.IApp, args ...string) error {
	return m(app, args...)
}

func (m quietMinibufferFn) OfferCompletion() bool {
	return false
}

func (m quietMinibufferFn) Arguments([]string) []minibuffer.IArg {
	return nil
}

//======================================================================

type substrArg struct {
	candidates []string
	sub        string
}

var _ minibuffer.IArg = substrArg{}

func (s substrArg) OfferCompletion() bool {
	return true
}

// return these in sorted order
func (s substrArg) Completions() []string {
	res := make([]string, 0)
	for _, str := range s.candidates {
		if strings.Contains(str, s.sub) {
			res = append(res, str)
		}
	}
	return res
}

//======================================================================

func newBoolArg(sub string) substrArg {
	return substrArg{
		sub:        sub,
		candidates: []string{"false", "true"},
	}
}

func newOnOffArg(sub string) substrArg {
	return substrArg{
		sub:        sub,
		candidates: []string{"off", "on"},
	}
}

func newSetArg(sub string) substrArg {
	return substrArg{
		sub: sub,
		candidates: []string{
			"auto-scroll",
			"copy-command-timeout",
			"dark-mode",
			"disable-shark-fin",
			"packet-colors",
			"pager",
			"nopager",
			"term",
			"noterm",
		},
	}
}

func newHelpArg(sub string) substrArg {
	return substrArg{
		sub: sub,
		candidates: []string{
			"cmdline",
			"map",
			"set",
			"vim",
		},
	}
}

//======================================================================

type unhelpfulArg struct {
	arg string
}

var _ minibuffer.IArg = unhelpfulArg{}

func (s unhelpfulArg) OfferCompletion() bool {
	return false
}

// return these in sorted order
func (s unhelpfulArg) Completions() []string {
	return nil
}

//======================================================================

type fileArg struct {
	substr string
}

var _ minibuffer.IArg = fileArg{}

func (s fileArg) OfferCompletion() bool {
	return true
}

func (s fileArg) Completions() []string {
	matches, _ := filepath.Glob(s.substr + "*")
	if matches == nil {
		return []string{}
	}
	return matches
}

//======================================================================

type recentsArg struct {
	substr string
}

var _ minibuffer.IArg = recentsArg{}

func (s recentsArg) OfferCompletion() bool {
	return true
}

func (s recentsArg) Completions() []string {
	matches := make([]string, 0)
	cfiles := termshark.ConfStringSlice("main.recent-files", []string{})
	if cfiles != nil {
		for _, sc := range cfiles {
			scopy := sc
			if strings.Contains(scopy, s.substr) {
				matches = append(matches, scopy)
			}
		}
	}

	return matches
}

//======================================================================

type filterArg struct {
	substr string
}

var _ minibuffer.IArg = filterArg{}

func (s filterArg) OfferCompletion() bool {
	return true
}

func (s filterArg) Completions() []string {
	matches := make([]string, 0)
	cfiles := termshark.ConfStringSlice("main.recent-filters", []string{})
	if cfiles != nil {
		for _, sc := range cfiles {
			scopy := sc
			if strings.Contains(scopy, s.substr) {
				matches = append(matches, scopy)
			}
		}
	}

	return matches
}

//======================================================================

type themeArg struct {
	substr string
}

var _ minibuffer.IArg = themeArg{}

func (s themeArg) OfferCompletion() bool {
	return true
}

func (s themeArg) Completions() []string {
	matches := make([]string, 0)

	// First gather built-in themes
	statikFS, err := fs.New()
	if err == nil {
		dir, err := statikFS.Open("/themes")
		if err == nil {
			info, err := dir.Readdir(-1)
			if err == nil {
				for _, finfo := range info {
					m := strings.TrimSuffix(finfo.Name(), ".toml")
					if strings.Contains(m, s.substr) {
						matches = append(matches, m)
					}
				}
			}
		}
	}

	// Then from filesystem
	stdConf := configdir.New("", "termshark")
	conf := stdConf.QueryFolderContainsFile("themes")
	if conf != nil {
		files, err := ioutil.ReadDir(filepath.Join(conf.Path, "themes"))
		if err == nil {
			for _, file := range files {
				m := strings.TrimSuffix(file.Name(), ".toml")
				if !termshark.StringInSlice(m, matches) {
					if strings.Contains(m, s.substr) {
						matches = append(matches, m)
					}
				}
			}
		}
	}

	return matches
}

//======================================================================

func stringIn(s string, a []string) bool {
	for _, s2 := range a {
		if s == s2 {
			return true
		}
	}
	return false
}

func parseOnOff(str string) (bool, error) {
	switch str {
	case "on", "ON", "On":
		return true, nil
	case "off", "OFF", "Off":
		return false, nil
	}
	return false, strconv.ErrSyntax
}

func validateTerm(term string) error {
	var err error
	_, err = terminfo.LookupTerminfo(term)
	if err != nil {
		_, _, err = dynamic.LoadTerminfo(term)
	}
	return err
}

type setCommand struct{}

var _ minibuffer.IAction = setCommand{}

func (d setCommand) Run(app gowid.IApp, args ...string) error {
	var err error
	var b bool
	var i uint64
	switch len(args) {
	case 3:
		switch args[1] {
		case "auto-scroll":
			if b, err = parseOnOff(args[2]); err == nil {
				AutoScroll = b
				termshark.SetConf("main.auto-scroll", AutoScroll)
				OpenMessage(fmt.Sprintf("Packet auto-scroll is now %s", gwutil.If(b, "on", "off").(string)), appView, app)
			}
		case "copy-timeout":
			if i, err = strconv.ParseUint(args[2], 10, 32); err == nil {
				termshark.SetConf("main.copy-command-timeout", i)
				OpenMessage(fmt.Sprintf("Copy timeout is now %ds", i), appView, app)
			}
		case "dark-mode":
			if b, err = parseOnOff(args[2]); err == nil {
				DarkMode = b
				termshark.SetConf("main.dark-mode", DarkMode)
			}
		case "disable-shark-fin":
			if b, err = strconv.ParseBool(args[2]); err == nil {
				termshark.SetConf("main.disable-shark-fin", DarkMode)
				OpenMessage(fmt.Sprintf("Shark-saver is now %s", gwutil.If(b, "off", "on").(string)), appView, app)
			}
		case "packet-colors":
			if b, err = parseOnOff(args[2]); err == nil {
				PacketColors = b
				termshark.SetConf("main.packet-colors", PacketColors)
				OpenMessage(fmt.Sprintf("Packet colors are now %s", gwutil.If(b, "on", "off").(string)), appView, app)
			}
		case "term":
			if err = validateTerm(args[2]); err == nil {
				termshark.SetConf("main.term", args[2])
				OpenMessage(fmt.Sprintf("Terminal type is now %s\n(Requires restart)", args[2]), appView, app)
			}
		case "pager":
			termshark.SetConf("main.pager", args[2])
			OpenMessage(fmt.Sprintf("Pager is now %s", args[2]), appView, app)
		default:
			err = invalidSetCommandErr
		}
	case 2:
		switch args[1] {
		case "noterm":
			termshark.DeleteConf("main.term")
			OpenMessage("Terminal type is now unset\n(Requires restart)", appView, app)
		case "nopager":
			termshark.DeleteConf("main.pager")
			OpenMessage("Pager is now unset", appView, app)
		default:
			err = invalidSetCommandErr
		}
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d setCommand) OfferCompletion() bool {
	return true
}

func (d setCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	res = append(res, newSetArg(toks[0]))

	if len(toks) > 0 {
		onOffCmds := []string{"auto-scroll", "dark-mode", "packet-colors"}
		boolCmds := []string{"disable-shark-fin"}
		intCmds := []string{"disk-cache-size-mb", "copy-command-timeout"}

		pref := ""
		if len(toks) > 1 {
			pref = toks[1]
		}

		if stringIn(toks[0], boolCmds) {
			res = append(res, newBoolArg(pref))
		} else if stringIn(toks[0], intCmds) {
			res = append(res, unhelpfulArg{})
		} else if stringIn(toks[0], onOffCmds) {
			res = append(res, newOnOffArg(pref))
		}
	}

	return res
}

//======================================================================

type readCommand struct {
	complete bool
}

var _ minibuffer.IAction = readCommand{}

func (d readCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	if len(args) != 2 {
		err = invalidReadCommandErr
	} else {
		RequestLoadPcapWithCheck(args[1], FilterWidget.Value(), app)
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d readCommand) OfferCompletion() bool {
	return d.complete
}

func (d readCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	pref := ""
	if len(toks) > 0 {
		pref = toks[0]
	}
	res = append(res, fileArg{substr: pref})
	return res
}

//======================================================================

type recentsCommand struct{}

var _ minibuffer.IAction = recentsCommand{}

func (d recentsCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	if len(args) != 2 {
		err = invalidRecentsCommandErr
	} else {
		RequestLoadPcapWithCheck(args[1], FilterWidget.Value(), app)
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d recentsCommand) OfferCompletion() bool {
	return true
}

func (d recentsCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	pref := ""
	if len(toks) > 0 {
		pref = toks[0]
	}
	res = append(res, recentsArg{substr: pref})
	return res
}

//======================================================================

type filterCommand struct{}

var _ minibuffer.IAction = filterCommand{}

func (d filterCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	if len(args) != 2 {
		err = invalidFilterCommandErr
	} else {
		setFocusOnDisplayFilter(app)
		FilterWidget.SetValue(args[1], app)
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d filterCommand) OfferCompletion() bool {
	return true
}

func (d filterCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	pref := ""
	if len(toks) > 0 {
		pref = toks[0]
	}
	res = append(res, filterArg{substr: pref})
	return res
}

//======================================================================

type themeCommand struct{}

var _ minibuffer.IAction = themeCommand{}

func (d themeCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	if len(args) != 2 {
		err = invalidThemeCommandErr
	} else {
		termshark.SetConf("main.theme", args[1])
		theme.Load(args[1])
		SetupColors()
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d themeCommand) OfferCompletion() bool {
	return true
}

func (d themeCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	pref := ""
	if len(toks) > 0 {
		pref = toks[0]
	}
	res = append(res, themeArg{substr: pref})
	return res
}

//======================================================================

type mapCommand struct {
	w *mapkeys.Widget
}

var _ minibuffer.IAction = mapCommand{}

func (d mapCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	if len(args) == 3 {
		key1 := vim.VimStringToKeys(args[1])
		if len(key1) != 1 {
			err = fmt.Errorf("Invalid: first map argument must be a single key (got '%s')", args[1])
		} else {
			keys2 := vim.VimStringToKeys(args[2])
			termshark.AddKeyMapping(termshark.KeyMapping{From: key1[0], To: keys2})
			mappings := termshark.LoadKeyMappings()
			for _, mapping := range mappings {
				d.w.AddMapping(mapping.From, mapping.To, app)
			}
		}
	} else if len(args) == 1 {
		OpenTemplatedDialogExt(appView, "Key Mappings", fixed, ratio(0.6), app)
	} else {
		err = invalidMapCommandErr
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d mapCommand) OfferCompletion() bool {
	return true
}

func (d mapCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	if len(toks) == 2 {
		res = append(res, unhelpfulArg{}, unhelpfulArg{})
	}
	return res
}

//======================================================================

type unmapCommand struct {
	w *mapkeys.Widget
}

var _ minibuffer.IAction = unmapCommand{}

func (d unmapCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	if len(args) != 2 {
		err = invalidMapCommandErr
	} else {
		key1 := vim.VimStringToKeys(args[1])
		d.w.ClearMappings(app)
		termshark.RemoveKeyMapping(key1[0])
		mappings := termshark.LoadKeyMappings()
		for _, mapping := range mappings {
			d.w.AddMapping(mapping.From, mapping.To, app)
		}
	}

	if err != nil {
		OpenMessage(fmt.Sprintf("Error: %s", err), appView, app)
	}

	return err
}

func (d unmapCommand) OfferCompletion() bool {
	return true
}

func (d unmapCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	res = append(res, unhelpfulArg{})
	return res
}

//======================================================================

type helpCommand struct{}

var _ minibuffer.IAction = helpCommand{}

func (d helpCommand) Run(app gowid.IApp, args ...string) error {
	var err error

	switch len(args) {
	case 2:
		switch args[1] {
		case "cmdline":
			OpenTemplatedDialog(appView, "CmdLineHelp", app)
		case "map":
			OpenTemplatedDialog(appView, "MapHelp", app)
		case "set":
			OpenTemplatedDialog(appView, "SetHelp", app)
		default:
			OpenTemplatedDialog(appView, "VimHelp", app)
		}
	default:
		OpenTemplatedDialog(appView, "UIHelp", app)
	}

	return err
}

func (d helpCommand) OfferCompletion() bool {
	return true
}

func (d helpCommand) Arguments(toks []string) []minibuffer.IArg {
	res := make([]minibuffer.IArg, 0)
	if len(toks) == 1 {
		res = append(res, newHelpArg(toks[0]))
	}
	return res
}

//======================================================================
// Local Variables:
// mode: Go
// fill-column: 110
// End:
