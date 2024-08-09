
package main

import (
	"slices"
	"strings"
	"strconv"
	"unicode"
	"log"
	"syscall"
	"time"
	"sync"

	"github.com/gdamore/tcell/v2"
)



// Helpers...
//
func Style2TcellStyle(style Style, base_style ...tcell.Style) tcell.Style {
	base := tcell.StyleDefault
	if len(base_style) != 0 {
		base = base_style[0] } 
	// set flags...
	colors := []string{}
	for _, s := range style {
		switch s {
			case "blink":
				base = base.Blink(true)
			case "bold":
				base = base.Bold(true)
			case "dim":
				base = base.Dim(true)
			case "italic":
				base = base.Italic(true)
			case "normal":
				base = base.Normal()
			case "reverse":
				base = base.Reverse(true)
			case "strike-through":
				base = base.StrikeThrough(true)
			case "underline":
				base = base.Underline(true)
			default:
				// urls...
				if string(s[:len("url")]) == "url" {
					p := strings.SplitN(s, ":", 2)
					url := ""
					if len(p) > 1 {
						url = p[1] }
					base = base.Url(url)
				// colors...
				} else {
					colors = append(colors, s) } } }
	// set the colors...
	if len(colors) > 0 && 
			colors[0] != "as-is" {
		base = base.Foreground(
			tcell.GetColor(colors[0])) }
	if len(colors) > 1 &&
			colors[1] != "as-is" {
		base = base.Background(
			tcell.GetColor(colors[1])) }
	return base }
func TcellEvent2Keys(evt tcell.EventKey) [][]string {
	mods := []string{}
	shifted := false

	var key, Key string

	mod, k, r := evt.Modifiers(), evt.Key(), evt.Rune()

	// handle key and shift state...
	if k == tcell.KeyRune {
		if unicode.IsUpper(r) {
			shifted = true
			Key = string(r)
			mods = append(mods, "shift") }
		key = string(unicode.ToLower(r))
	// special keys...
	} else if k > tcell.KeyRune || k <= tcell.KeyDEL {
		key = evt.Name()
	// ascii...
	} else {
		if unicode.IsUpper(rune(k)) {
			shifted = true 
			Key = string(rune(k))
			mods = append(mods, "shift") } 
		key = strings.ToLower(string(rune(k))) } 

	// split out mods and normalize...
	key_mods := strings.Split(key, "+")
	key = key_mods[len(key_mods)-1]
	if k := []rune(key) ; len(k) == 1 && unicode.IsUpper(k[0]) {
		key = strings.ToLower(key) }
	key_mods = key_mods[:len(key_mods)-1]

	// basic translation...
	if key == " " {
		key = "Space" }

	if slices.Contains(key_mods, "Ctrl") || 
			mod & tcell.ModCtrl != 0 {
		mods = append(mods, "ctrl") }
	if slices.Contains(key_mods, "Alt") || 
			mod & tcell.ModAlt != 0 {
		mods = append(mods, "alt") }
	if slices.Contains(key_mods, "Meta") || 
			mod & tcell.ModMeta != 0 {
		mods = append(mods, "meta") }
	if !shifted && mod & tcell.ModShift != 0 {
		mods = append(mods, "shift") }

	return key2keys(mods, key, Key) }
func TcellEvent2Mouse(evt tcell.EventMouse) [][]string {
	mods := []string{}
	mod, b := evt.Modifiers(), evt.Buttons()

	// buttons...
	// XXX do we handle any other keys???
	var button string
	if b & tcell.Button1 != 0 {
		button = "MouseLeft"
	} else if b & tcell.Button2 != 0 {
		button = "MouseRight" 
	} else if b & tcell.Button2 != 0 {
		button = "MouseMiddle" 
	} else if b & tcell.WheelUp != 0 {
		button = "WheelUp"
	} else if b & tcell.WheelDown != 0 {
		button = "WheelDown" 
	} else {
		button = "MouseHover" }

	// split out mods and normalize...
	if mod & tcell.ModCtrl != 0 {
		mods = append(mods, "ctrl") }
	if mod & tcell.ModAlt != 0 {
		mods = append(mods, "alt") }
	if mod & tcell.ModMeta != 0 {
		mods = append(mods, "meta") }
	if mod & tcell.ModShift != 0 {
		mods = append(mods, "shift") }

	return key2keys(mods, button) }



type Tcell struct {
	tcell.Screen `no-flag:"true"`

	Lines *Lines `no-flag:"true"`

	// XXX this is not seen by flags...
	FocusAction bool `long:"focus-action" description:"if not set the focusing click will be ignored"`

	// caches...
	// NOTE: in normal use-cases the stuff cached here is static and 
	//		there should never be any leakage, if there is then something 
	//		odd is going on.
	__style_cache map[string]tcell.Style
	__updating_style_cache sync.Mutex
}

func (this *Tcell) ResetCache() {
	this.__style_cache = nil }

// Extends Style2TcellStyle(..) by adding cache...
//
// XXX do we need this public???
// XXX URLS are supported but not usable yet as there is no way to set 
//		the url...
//		use: "url:<url>"
// XXX would be nice to be able to use "foreground" and "background" 
//		colors in a predictable manner -- currently they reference curent 
//		colors
//		...i.e. {"yellow", "foreground"} will set both colors to yellow...
// XXX need a way to get default style without .Lines.GetStyle(..)...
func (this *Tcell) style2TcellStyle(style_name string, style Style) tcell.Style {
	this.__updating_style_cache.Lock()
	defer this.__updating_style_cache.Unlock()
	// cache...
	if this.__style_cache == nil {
		this.__style_cache = map[string]tcell.Style{} }
	s, ok := this.__style_cache[style_name]
	if ok {
		return s }
	cache := func(s tcell.Style) tcell.Style {
		this.__style_cache[style_name] = s 
		return s }

	// base style (cached manually)...
	base, ok := this.__style_cache["default"]
	if ! ok {
		// XXX need to move this out of here...
		_, s := this.Lines.GetStyle("default")
		base = Style2TcellStyle(s) 
		this.__style_cache["default"] = base }

	return cache(
		Style2TcellStyle(style, base)) }

var ANSI_SGR = map[string]string{
	// XXX
}
// 3-4 but color...
var ANSI_COLOR = map[string]tcell.Color {
	"0": tcell.GetColor("black"),
	"1": tcell.GetColor("red"),
	"2": tcell.GetColor("green"),
	"3": tcell.GetColor("yellow"),
	"4": tcell.GetColor("blue"),
	"5": tcell.GetColor("magenta"),
	"6": tcell.GetColor("cyan"),
	"7": tcell.GetColor("white"),
}
var ANSI_COLOR_PREFIX = map[string]string{
	"3": "fg",
	"4": "bg",
	"9": "fg-bright",
	"10": "bg-bright",
}
// XXX SGR this is not done yet...
func ansi2style(style tcell.Style, parts ...string) tcell.Style {
	if parts[0] == "38" || parts[0] == "48" {
		var color tcell.Color
		atoi := func(s string) int {
			i, err := strconv.Atoi(s)
			if err != nil {
				log.Println("ansi2style(..): can't parse color:", s) }
			return i }
		// palette / 8-bit...
		if parts[1] == "5" {
			color = tcell.PaletteColor(
				atoi(parts[2]))
		// RGB / 24-bit...
		} else if parts[1] == "2" {
			color = tcell.NewRGBColor(
				int32(atoi(parts[2])), 
				int32(atoi(parts[3])), 
				int32(atoi(parts[4]))) }
		if parts[0] == "38" {
			style = style.Foreground(color)
		} else if parts[0] == "48" {
			style = style.Background(color) }
	// 4-bit...
	} else {
		for _, p := range parts {
			if p == "00" {
				style = style.Normal()
			} else if p == "01" {
				style = style.Bold(true)
			// param...
			} else if x, ok := ANSI_SGR[p]; ok {
				// XXX see: https://en.wikipedia.org/wiki/ANSI_escape_code
				log.Println("SGR:", x)
			// color...
			} else {
				color := ANSI_COLOR[string(p[len(p)-1])]
				prefix := strings.Split(ANSI_COLOR_PREFIX[string(p[:len(p)-1])], "-")
				if len(prefix) > 1 && 
						string(prefix[1]) == "bright" {
					color += 8 }
				if string(prefix[0]) == "fg" {
					style = style.Foreground(color)
				} else {
					style = style.Background(color) } } } }
	return style }

func (this *Tcell) drawCells(col, row int, str string, style_name string, style Style) int {
	if style_name == "EOL" {
		return 0 }
	c := 0
	base := this.style2TcellStyle(style_name, style)
	cur := base
	runes := []rune(str)
	offset := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		// handle escape sequences/styles...
		if r == '\x1B' {
			seq := CollectANSIEscSeq(runes, i)
			// XXX handle colors...
			if seq[len(seq)-1] == 'm' {
				c := strings.Split(string(seq[2:len(seq)-1]), ";")
				// reset...
				if len(c) == 1 && c[0] == "0" {
					cur = base
				// parse color...
				} else if len(c) == 2 {
					cur = ansi2style(base, c...) } }
			offset += len(seq)
			i += len(seq) - 1 
			continue }
		this.SetContent(col+i-offset, row, r, nil, cur)
		c++ }
	return c }

func (this *Tcell) Fill(style Style) {
	this.Screen.Fill(' ', this.style2TcellStyle("background", style)) }
func (this *Tcell) Refresh() {
	this.Screen.Sync()
	this.Screen.Show() }

func (this *Tcell) Setup(lines *Lines) {
	this.Lines = lines
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Panic(err) }
	this.Screen = screen }

func (this *Tcell) Init() {
	if err := this.Screen.Init(); err != nil {
		log.Panic(err) }
	this.EnableMouse()
	this.EnablePaste()
	this.EnableFocus() }
// XXX can we detect mod key press???
//		...need to detect release of shift in selection...
// XXX add background fill...
// XXX might be fun to indirect this, i.e. add a global workspace manager
//		that would pass events to clients/windows and handle their draw 
//		order...
// XXX focus handling is not 100% yet -- revise if it's better to remove it?
func (this *Tcell) Loop(ui *UI) Result {
	defer this.Finalize()
	this.Init()

	// initial state...
	ui.
		updateGeometry().
		Draw()

	just_started := true
	skip := false
	for {
		this.Show()

		evt := this.PollEvent()

		switch evt := evt.(type) {
			// resize...
			case *tcell.EventResize:
				skip = false
				ui.
					updateGeometry().
					Draw()
			// focus...
			// XXX this behaves erratically...
			//		...seems that if the switch is too fast skipping will not work...
			// XXX is the focus event guaranteed to precede the click 
			//		that focused the window??
			case *tcell.EventFocus:
				//log.Println("Focus:", focus)
				//focus := evt.Focused
				// skip first mouse interaction after focus...
				// NOTE: we are ignoring focus change on startup (just_started)...
				if ! just_started && 
						! this.FocusAction {
					skip = true }
				just_started = false
			// keys...
			case *tcell.EventKey:
				skip = false
				key_handled := false
				for _, key := range TcellEvent2Keys(*evt) {
					res := ui.HandleKey(strings.Join(key, "+"))
					if res == Skip {
						continue }
					if res == Missing {
						//log.Println("Key Unhandled:", key)
						continue }
					if res != OK {
						return res } 
					key_handled = true
					ui.Draw()
					break }
				// do not check for defaults on keys we handled...
				if key_handled {
					continue }
				// defaults...
				if evt.Key() == tcell.KeyEscape || 
						evt.Key() == tcell.KeyCtrlC {
					return OK }
			// mouse...
			case *tcell.EventMouse:
				if skip {
					skip = false
					continue }
				col, row := evt.Position()
				for _, key := range TcellEvent2Mouse(*evt) {
					res := ui.HandleMouse(col, row, key)
					if res == Skip {
						continue }
					if res == Missing {
						//log.Println("Key Unhandled:", key)
						continue }
					if res != OK {
						return res } 
					ui.Draw()
					break } } }
	return OK }
func (this *Tcell) Stop() {
	screen := this.Screen
	_, ok := screen.Tty()
	if ! ok {
		return }
	// XXX can we go around all of this and simply pass ctrl-z to parent???
	screen.Suspend()
	pid := syscall.Getppid()
	// ask parent to detach us from io...
	err := syscall.Kill(pid, syscall.SIGSTOP)
	if err != nil {
		log.Panic(err) }
	// stop...
	err = syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	if err != nil {
		log.Panic(err) }
	time.Sleep(time.Millisecond*50)
	screen.Resume() }
// handle panics and cleanup...
func (this *Tcell) Finalize() {
	maybePanic := recover()
	this.Screen.Fini()
	if maybePanic != nil {
		panic(maybePanic) } }



// vim:set sw=4 ts=4 nowrap :
