
package main

import (
	"slices"
	"strings"
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
	__updating_cache sync.Mutex
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
	// cache...
	if this.__style_cache == nil {
		this.__updating_cache.Lock()
		this.__style_cache = map[string]tcell.Style{}
		this.__updating_cache.Unlock() }
	s, ok := this.__style_cache[style_name]
	if ok {
		return s }
	cache := func(s tcell.Style) tcell.Style {
		this.__updating_cache.Lock()
		defer this.__updating_cache.Unlock()
		this.__style_cache[style_name] = s 
		return s }

	// base style (cached manually)...
	base, ok := this.__style_cache["default"]
	if ! ok {
		// XXX need to move this out of here...
		_, s := this.Lines.GetStyle("default")
		base = Style2TcellStyle(s) 
		this.__updating_cache.Lock()
		this.__style_cache["default"] = base
		this.__updating_cache.Unlock() }

	return cache(
		Style2TcellStyle(style, base)) }
func (this *Tcell) drawCells(col, row int, str string, style_name string, style Style){
	if style_name == "EOL" {
		return }
	s := this.style2TcellStyle(style_name, style)
	for i, r := range []rune(str) {
		this.SetContent(col+i, row, r, nil, s) } }

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
