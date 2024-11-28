
package main

import (
	"fmt"
	"log"
	//"io"
	"strings"
	"strconv"
	"slices"
	//"bufio"
	"sync"
	"regexp"
	"os"
	"time"
)



// ANSI escape codes...
//

// Collect escape sequence...
//
// see: 
//	https://en.wikipedia.org/wiki/ANSI_escape_code	
//	https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797 
func CollectANSIEscSeq(runes []rune, i int) []rune {
	if len(runes) == 0 || 
			i >= len(runes) ||
			runes[i] != '\x1B' {
		return []rune{} }
	runes = runes[i:]
	// collect the sequence...
	seq := []rune{ runes[0] }
	commands := "HfABCDEFGnsuJKmhlp"
	if runes[1] != '[' {
		commands = "M78" }
	i = 1
	for i < len(runes) { 
		seq = append(seq, runes[i]) 
		if strings.ContainsRune(commands, runes[i]) {
			break }
		i++ } 
	return seq }

// Strip out escape sequences...
//
func StripANSIEscSeq(runes []rune) []rune {
	output := slices.Grow([]rune{}, len(runes))
	i := 0
	for ; i < len(runes); i++ {
		if runes[i] == '\x1B' {
			seq := CollectANSIEscSeq(runes, i)
			i += len(seq)-1 
			continue }
		output = append(output, runes[i]) }
	return output }



// Spinner...
//
type Spinner struct {
	Frames string `long:"spinner" value-name:"THEME|STR" default:"><" env:"SPINNER" description:"Spinner frames"`
	State int

	running int
	starting sync.Mutex
	tick sync.Mutex

	interval time.Time
}
func (this *Spinner) String() string {
	this.tick.Lock()
	defer this.tick.Unlock()
	if this.running <= 0 {
		return "" } 
	frames := this.Frames
	if frames == "" {
		frames = SPINNER_THEME[SPINNER_DEFAULT] }
	return string([]rune(frames)[this.State]) }
func (this *Spinner) Start() {
	this.starting.Lock()
	defer this.starting.Unlock()
	this.running++ 
	if this.running > 1 {
		return }
	if this.State < 0 {
		this.Step() }
	go func(){
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			if this.running <= 0 {
				return }
			this.Step() } }() }
func (this *Spinner) Stop() *Spinner {
	if this.running == 1 {
		return this.StopAll() }
	if this.running > 0 {
		this.running-- }
	return this }
func (this *Spinner) StopAll() *Spinner {
	if this.running > 0 {
		this.running = 0
		//ACTIONS.Refresh() 
	}
	return this }
// XXX should this draw the whole screen???
//		...might be nice to be able to only update the chrome (title/status)
func (this *Spinner) Step() string {
	this.tick.Lock()
	if this.running <= 0 {
		return "" }
	frames := this.Frames
	if frames == "" {
		frames = SPINNER_THEME[SPINNER_DEFAULT] }
	this.State++
	if this.State >= len([]rune(frames)) {
		this.State = 0 }
	this.tick.Unlock()
	// XXX should this draw the whole screen???
	//ACTIONS.Refresh()
	return this.String() }
func (this *Spinner) Done() *Spinner {
	this.StopAll()
	return this }



// Theme...
//
type Style []string
type Theme map[string]Style
var THEME = Theme {
	// NOTE: this is used as basis for all other styles.
	"default": {},
	/*/
	"default": {
		"white",
		"darkblue",
	},
	//*/

	"normal-text": {},
	"normal-separator": {
		"gray", 
	},
	"normal-overflow": {
		"lightgray", 
	},

	"selected-text": {
		"bold",
		"yellow", 
	},
	"selected-separator": {
		"gray", 
	},
	"selected-overflow": {
		"lightgray", 
	},

	"current-text": {
		"bold",
		"reverse",
	},
	"current-separator": {
		"reverse",
		// color placeholder -- use color "as-is"...
		"as-is",
		"gray", 
	},
	"current-overflow": {
		"reverse",
		"as-is",
		"gray", 
	},

	"current-selected-text": {
		"bold",
		"reverse",
		"yellow", 
	},
	"current-selected-separator": {
		"gray",
		"yellow", 
	},
	"current-selected-overflow": {
		"gray",
		"yellow", 
	},

	"title": {
		"bold",
	},
	/*
	"title-text": {},
	"title-separator": {},
	"title-overflow": {},
	//*/

	"status": {},
	/*
	"status-text": {},
	"status-separator": {},
	"status-overflow": {},
	//*/

	"background": {},
	"border": {},
}
// Get best matching theme...
//
// Search example:
//		theme.GetStyle("current-selected-text")
//			-> theme["current-selected-text"]?
//			-> theme["current-selected"]?
//			-> theme["current-text"]?
//			-> theme["current"]?
//			-> theme["selected-text"]?
//			-> theme["selected"]?
//			-> theme["default-text"]?
//			-> theme["default"]?
//			-> {}
func (this Theme) GetStyle(style string) (string, Style) {
	// special case...
	if style == "EOL" {
		return "EOL", []string{"EOL"} }
	// direct match...
	res, ok := this[style]
	if ok {
		return style, res }
	var get func([]string) (string, Style)
	get = func(s []string) (string, Style) {
		if len(s) == 1 {
			return "", []string{} }
		n := strings.Join(s[:len(s)-1], "-")
		res, ok := this[n]
		if ok {
			return n, res } 
		if len(s) > 2 {
			if n, res := get(append(s[:len(s)-2], s[len(s)-1])) ; len(res) > 0 {
				return n, res } 
			if n, res := get(s[1:]) ; len(res) > 0 {
				return n, res } }
		return "", []string{} }
	// search...
	s := strings.Split(style, "-")
	if n, res := get(s); len(res) > 0 {
		return n, res }
	// default...
	if len(s) > 1 {
		res, ok := this["default-"+ s[len(s)-1]]
		if ok {
			return "default-"+ s[len(s)-1], res } }
	res, ok = this["default"]
	if ok {
		return "default", res }
	return "default", []string{} }

var BORDER_DEFAULT = "single"
var BORDER_THEME = map[string]string {
	"single": "│┌─┐│└─┘",
	"thick": "┃┏━┓┃┗━┛",
	"double": "║╔═╗║╚═╝",
	"mixed": "│┌─┒┃┕━┛",
	"mixed-double": "│┌─╖║╘═╝",
	"single-double": "│╒═╕│╘═╛",
	"double-single": "║╓─╖║╙─╜",
	"shaded": "│┌─┐┃└━┛",
	"shaded-double": "│┌─┐║└═╝",
	"ascii": "|+-+|+-+",
}

var SPINNER_DEFAULT = "><"
var SPINNER_THEME = map[string]string {
	"><": "><",
	"rotating-v": "v<^>",
	// NOTE: can't use "|" (span marker), thus this uses a unicode analogue...
	"rotating-line": "-\\❘/",
	"dots-spin": "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏",
	"dot-jump": "⠁⠂⠄⡀⢀⠠⠐⠈",
	"bin-counter": "⡀⡁⡂⡃⡄⡅⡆⡇⡈⡉⡊⡋⡌⡍⡎⡏⡐⡑⡒⡓⡔⡕⡖⡗⡘⡙⡚⡛⡜⡝⡞⡟⡠⡡⡢⡣⡤⡥⡦⡧⡨⡩⡪⡫⡬⡭⡮⡯⡰⡱⡲⡳⡴⡵⡶⡷⡸⡹⡺⡻⡼⡽⡾⡿⢀⢁⢂⢃⢄⢅⢆⢇⢈⢉⢊⢋⢌⢍⢎⢏⢐⢑⢒⢓⢔⢕⢖⢗⢘⢙⢚⢛⢜⢝⢞⢟⢠⢡⢢⢣⢤⢥⢦⢧⢨⢩⢪⢫⢬⢭⢮⢯⢰⢱⢲⢳⢴⢵⢶⢷⢸⢹⢺⢻⢼⢽⢾⢿⣀⣁⣂⣃⣄⣅⣆⣇⣈⣉⣊⣋⣌⣍⣎⣏⣐⣑⣒⣓⣔⣕⣖⣗⣘⣙⣚⣛⣜⣝⣞⣟⣠⣡⣢⣣⣤⣥⣦⣧⣨⣩⣪⣫⣬⣭⣮⣯⣰⣱⣲⣳⣴⣵⣶⣷⣸⣹⣺⣻⣼⣽⣾⣿",
	"dots": "⣾⣽⣻⢿⡿⣟⣯⣷",
	"dots-eight": "⠋⠙⠚⠒⠂⠂⠒⠲⠴⠦⠖⠒⠐⠐⠒⠓⠋",
	"dots-jump": "⢄⢂⢁⡁⡈⡐⡠",
	"lines": "☱☲☴☲",
	"blink-rombus": "◇◈◆",
	"blink-square": "■□▪▫",
	"squares": "◰◳◲◱",
	"circle-half": "◐◓◑◒",
	"circle-quarter": "◴◷◶◵",
	"block-spin": "▌▀▐▄",
	"blocks-turn": "▖▘▝▗",
	"line-flip": "┤┘┴└├┌┬┐",
}



type Env map[string]string



// CellsDrawer
//
// XXX not sure how to define an easily overloadable/extendable "object"... 
//		...don't tell me that a Go-y solution is passing function pointers))))
// XXX revise name... 
type CellsDrawer interface {
	drawCells(col, row int, str string, style_name string, style Style) int
}



// Lines
//

var TAB_SIZE = 8

var OVERFLOW_INDICATOR = '}'

var SPAN_MARKER = "|"
// XXX this would require us to support escaping...
//var SPAN_MARKER = "|"
var SPAN_MIN_SIZE = 3

//var SCROLLBAR = "█░"
var SCROLLBAR = "┃│"

// XXX should this be Reader/Writer???
type Lines struct {
	CellsDrawer `no-flag:"true"`

	// XXX is this a good idea???
	LinesBuffer

	// template placeholder handlers...
	Placeholders *Placeholders

	// geometry...
	Top int
	Left int
	Width int
	Height int

	// positioning...
	RowOffset int
	ColOffset int

	Env Env

	//Theme Theme

	// chrome...
	Title string `long:"title" value-name:"TEXT" default:" $TEXT_LEFT |%F%S%F" env:"TITLE" description:"Title line"`
	TitleDisabled bool `long:"no-title" description:"Disable title line"`
	Status string `long:"status" value-name:"TEXT" default:"|${SELECTED:!*}${SELECTED:+($SELECTED)}%F $LINE/$LINES " env:"STATUS" description:"Status line"`
	StatusDisabled bool `long:"no-status" description:"Disable status line"`

	// Format: 
	//		"│┌─┐│└─┘"
	//		 01234567
	Border string `long:"border" value-name:"[THEME|STR]" env:"BORDER" default:"│┌─┐│└─┘" description:"Set border chars"`

	OverflowIndicator string `long:"overflow-indicator" value-name:"C" default:"}" description:"Overflow indicator char"`
	OverflowOverBorder bool

	// Format: 
	//		"█░"
	//		 01
	// NOTE: if this is set to "" the default SCROLLBAR will be used.
	Scrollbar string `long:"scrollbar" value-name:"STR" env:"SCROLLBAR" default:"┃│" description:"Set scrollbar chars"`
	ScrollbarDisabled bool

	Filler rune

	// column spanning...
	// NOTE: values of 0 are swapped for .SpanMinSize
	SpanMode string `long:"span" value-name:"[STR|fit-right|none]" env:"SPAN" description:"Span columns"`
	SpanModeTitle string `long:"span-title" value-name:"STR" env:"SPAN_TITLE" default:"*,3" description:"Span title columns"`
	SpanModeStatus string `long:"span-status" value-name:"STR" env:"SPAN_STATUS" description:"Span status columns"`
	// cache...
	// XXX do we need to cache multiple values???
	__SpanMode_cache struct {
		text string
		width int
		sep int
		value []int
	}
	SpanMarker string
	SpanSeparator string `long:"span-separator" value-name:"C" default:"│" env:"SPAN_SEP" description:"Span separator"`
	// defaults to: SPAN_MIN_SIZE
	// NOTE: this affects only % and * spans, explicit spans are not changed.
	SpanMinSize int `long:"span-min" value-name:"N" default:"8" env:"SPAN_MIN_SIZE" description:"Minimum span size"`
	SpanNoExtend bool

	Spinner Spinner

	ANSIEscapeSeq string `long:"ansi-seq" choice:"handle" choice:"hide" choice:"show" default:"handle" description:"Sets how ANSI Escape Sequances are handled"`

	TabSize int `long:"tab-size" value-name:"N" default:"8" env:"TAB" description:"Tab size"`

	Theme Theme `long:"theme" value-name:"NAME:[STYLE,]FGCOLOR[,BGCOLOR]" description:"Set theme color"`
}

func (this *Lines) Rows() int {
	h := this.Height
	if ! this.TitleDisabled || 
			this.Border != "" {
		h-- }
	if ! this.StatusDisabled || 
			this.Border != "" {
		h-- }
	return h }
func (this *Lines) Cols() int {
	w := this.Width
	// borders...
	if this.Border != "" {
		w -= 2
	// no borders + scrollbar...
	} else if ! this.ScrollbarDisabled && 
			//this.Rows() < len(this.Lines) {
			this.Rows() < this.Len() {
		w-- }
	return w }
func (this *Lines) Scrollable() bool {
	//return len(this.Lines) > this.Rows() }
	return this.Len() > this.Rows() }

func (this *Lines) GetStyle(style string) (string, Style) {
	theme := this.Theme
	if theme == nil {
		theme = THEME }
	return theme.GetStyle(style) }

// XXX for some reason overflow is triggered before the last char iin line in:
//			$ go run . -c 'ls --color=yes ~/Pictures/' -t "sed 's/$/|/'" 2> log || (sleep 5 && reset)
//		but if we simply echo something all is correct...
//		...is there a non-printable we missed???
// NOTE: tabs are always expanded with ' '...
func (this *Lines) makeSection(str string, width int, rest ...string) (string, bool) {
	fill := ' '
	if len(rest) >= 1 {
		fill = []rune(rest[0])[0]
	} else if this.Filler != 0 {
		fill = this.Filler }
	// defaults...
	tab := this.TabSize
	if tab == 0 {
		tab = TAB_SIZE }

	runes := []rune(str)
	//expand := true
	//if width < 0 {
	//	return "", true }
	if width == 0 {
		//expand = false
		width = len(runes) }

	// NOTE: this can legally get longer than width if it contains escape 
	//		sequeces (i.e. this.ANSIEscapeSeq != "hide")...
	output := []rune(strings.Repeat(" ", width))
	terminated := false
	i, o := 0, 0
	for ; o < len(output); i, o = i+1, o+1 {
		r := fill
		// get the rune...
		if i < len(runes) {
			r = runes[i] 
		// no more runes -> reset color...
		} else if ! terminated {
			terminated = true
			// reset color at end of line...
			s := []rune("\x1B[0m")
			output = slices.Insert(output, o, s...) 
			o += len(s) }

		switch r {
			// escape sequences...
			case '\x1B' :
				seq := CollectANSIEscSeq(runes, i)
				if this.ANSIEscapeSeq != "hide" {
					// show escape sequence...
					if this.ANSIEscapeSeq == "show" {
						r = '␛'
					// extend output to accommodate later removal of escape sequence...
					} else {
						output = append(output, []rune(strings.Repeat(" ", len(seq)))...) }
				// remove escape sequence...
				} else {
					i += len(seq) - 1
					o-- 
					continue }
			// tab -- offset output to next tabstop... 
			case '\t':
				d := tab - (o % tab) 
				o += d
				continue 
			// special chars...
			// XXX use a decode table...
			//case '\0' :
			//	r = '␀'
			case '\n' :
				r = '␤'
			case '\r' :
				r = '␍' }

		// set the rune...
		output[o] = r }
	
	return string(output), 
		// overflow...
		i < len(runes) }
func (this *Lines) parseSizes(str string, width int, sep int) []int {
	str = strings.TrimSpace(str)
	min_size := this.SpanMinSize
	if min_size == 0 {
		min_size = SPAN_MIN_SIZE }
	spec := []string{}

	// special case: single col...
	if str == "100%" || 
			str == "*" ||
			str == "" {
		return []int{width}
	// 2+ cols...
	} else {
		spec = strings.Split(str, ",") 
		// only one col specified -> append "*"...
		if len(spec) < 2 {
			spec = append(spec, "*") } }

	// parse list of sizes...
	rest := width
	sizes := []int{}
	stars := 0
	for i, size := range spec {
		size = strings.TrimSpace(size)
		cols := -1
		// *...
		// NOTE: *'s are merked as -1 width and expanded after we calculate 
		//		all the concrete sizes...
		if size == "*" || 
				size == "" {
			stars++
		// %...
		} else if size[len(size)-1] == '%' {
			p, err := strconv.ParseFloat(string(size[:len(size)-1]), 64)
			if err != nil {
				log.Println("Error parsing:", size, "in:", str) 
				stars++
				sizes = append(sizes, cols)
				continue }
			// XXX CEIL_ROUND the "+ 0.5" biases the rounding up (ceiling) 
			//		and fixes the "50%" -> 49/51 @ 101 split but will 
			//		this break other things???
			cols = int(float64(width) * (p / 100) + 0.5)
			// accout for separators...
			if i < len(spec)-1 {
				cols -= sep }
			// min width...
			if cols > 0 && 
					cols < min_size {
				cols = min_size }
		// explicit cols...
		// NOTE: these do not include separators...
		} else {
			var err error
			cols, err = strconv.Atoi(size) 
			if err != nil {
				log.Println("Error parsing:", size, "in:", str) 
				stars++
				sizes = append(sizes, cols)
				continue } 
			if cols < 0 {
				cols = min_size }
			if i < len(spec)-1 {
				rest -= sep } }
		if cols > 0 {
			rest -= cols }
		sizes = append(sizes, cols) }

	// precalculate * sizes...
	star_sizes := []int{}
	if stars > 0 {
		size := int(float64(rest) / float64(stars) + 0.5)
		if size < min_size {
			size = min_size }
		for i := 0; i < stars; i++ {
			star_sizes = append(star_sizes, size) }
		// overflowing/underflowing (rounding error)...
		if size > min_size {
			d := rest - size * stars
			// trim tail elements...
			if d < 0 {
				for i := stars-1; d < 0 && i >= 0; i--  {
					star_sizes[i]--
					d++ }
			// pad head elements...
			} else if d > 0 {
				for i := 0; d > 0 && i < stars; i++  {
					star_sizes[i]++
					d-- } } } }

	// fill *'s and trim overflow...
	star := 0
	total := 0
	for i := 0; i < len(sizes); i++ {
		// special case: overflow at the separator...
		if sep > 0 && 
				//i < len(sizes)-1 && 
				total == width {
			sizes[i] = 0
			total++
			continue }
		if total >= width {
			sizes[i] = -1
			continue }
		size := sizes[i]
		// *...
		if size < 0 {
			size = star_sizes[star]
			star++
			if i < len(sizes)-1 {
				size -= sep } }
		total += size 
		if i < len(sizes)-1 {
			total += sep }
		// overflow -- trim the cell...
		if total > width {
			size -= total - width
			// uncompensate for separator...
			if i < len(sizes)-1 {
				size += sep } }
		// underflow -- add excess to last cell...
		if i == len(sizes)-1 && 
				total < width {
			size += width - total }
		sizes[i] = size }
	return sizes }
func (this *Lines) splitSections(str string, split ...bool) []string {
	if len(split) > 0 && 
			! split[0] {
		return []string{ str } }
	marker := this.SpanMarker
	if marker == "" {
		marker = SPAN_MARKER }
	return strings.Split(str, marker) }
func (this *Lines) makeSections(str, span string, width int, sep_size int, rest ...string) []string {
	// defaults...
	split := true
	if span == "default" {
		span = this.SpanMode }
	if span == "none" {
		split = false
		span = "*" }
	overflow := string(OVERFLOW_INDICATOR)
	if len(this.OverflowIndicator) != 0 {
		overflow = string([]rune(this.OverflowIndicator)[0]) }

	sections := this.splitSections(str, split)

	skip := false
	doSection := func(str string, width int) []string {
		str, o := this.makeSection(str, width, rest...)
		sep := ""
		// mark overflow if skipping sections too...
		if o || skip {
			sep = overflow }
		return []string{ 
			str, 
			sep, 
		} }

	// single section...
	res := []string{}
	if len(sections) == 1 {
		return doSection(sections[0], width)

	} else {
		// sizing: automatic (uncached)...
		// NOTE: we do not cache this because the output depends on sections...
		sizes := []int{}
		if span == "" || span == "fit-right" {
			min_size := this.SpanMinSize
			if min_size == 0 {
				min_size = SPAN_MIN_SIZE }
			section := doSection(sections[len(sections)-1], 0)
			// XXX need to account for ANSI escape sequences...
			l := len([]rune(section[0]))
			if l <= 0 {
				l = min_size }
			sizes = this.parseSizes("*,"+ fmt.Sprint(l), width, sep_size)
		// sizing: manual (cached)...
		} else {
			// cached result -- the same for each line, no need to recalculate...
			if span == this.__SpanMode_cache.text && 
					this.__SpanMode_cache.sep == sep_size &&
					this.__SpanMode_cache.width == width {
				sizes = this.__SpanMode_cache.value
			// generate/cache...
			} else {
				sizes = this.parseSizes(span, width, sep_size)
				this.__SpanMode_cache.text = span
				this.__SpanMode_cache.width = width
				this.__SpanMode_cache.sep = sep_size
				this.__SpanMode_cache.value = sizes } } 
			//*/
		// build the sections...
		var i int
		getSection := func(i int) string {
			section := ""
			if i < len(sections) {
				section = sections[i] }
			return section }
		for i=0; i < len(sizes)-1; i++ {
			// overflow...
			if sizes[i] <= 0 {
				if len(res) > 0 {
					// a zero column -- separator + overflow...
					if sizes[i] == 0 {
						res = append(res, "") }
					res[len(res)-1] = overflow }
				break }
			res = append(res, doSection(getSection(i), sizes[i])...) }
		// last section...
		if sizes[i] == 0 {
			if len(res) == len(sizes) {
				res = append(res, overflow) }
		} else if sizes[i] > 0 {
			res = append(res, doSection(getSection(i), sizes[i])...) } }
	// minimal section...
	if len(res) == 0 {
		return []string{"", ""} }
	return res }
//
//	.makeSectionChrome(<str>, <span>, <width>[, <left_border>, <right_border>[, <filler>]])
//		-> <line>
//
// Format:
//		<line> ::=
//			<lborder> <term>
//			| <lborder> <sections> <text> <term>
//		<sections> ::= 
//			<empty>
//			| <section>
//			| <section> [ <res> ]
//		<section> ::= 
//			<text> <sep>
//		<term> ::=
//			<rborder>
//			| <overflow>
//
// NOTE: we are not joining the list here so as to enable further 
//		processing (e.g. styling) down the line...
// XXX should we be able to distinguish between last cell overflow and 
//		and section overflow???
//		...currently it is not possible to do so...
// XXX make sure to handle/trim lines ending in escape sequences correctly 
//		when embedding overflow indicator...
//		...trimming part of a sequence off is unlikely to be a big issue 
//		as we trimming e sequence will remove it's command and we 
//		ignore/remove unknown and malformed sequences...
func (this *Lines) makeSectionChrome(str, span string, width int, rest ...string) []string {
	separator := this.SpanSeparator
	if len(rest) >= 1 {
		separator = rest[0]
		rest = rest[1:] }
	border := this.Border
	if len(border) > 0 && 
			len(border) < 8 {
		border += strings.Repeat(" ", 8 - len(border)) }
	border_l := ""
	border_r := ""
	if len(rest) >= 2 {
		border_l = rest[0] 
		border_r = rest[1] 
		width -= len([]rune(border_l)) + len([]rune(border_r))
		rest = rest[2:]
	} else if border != "" {
		border_l = string([]rune(border)[0])
		border_r = string([]rune(border)[4])
		width -= 2 }
	if width < 0 {
		// XXX if we get here then something odd is going on...
		log.Fatal("ERR: makeSectionChrome(..): drawing zero or negative width")
		width = 0 }
	// XXX BUG: looks like this yiels the same results with sep_size of 0 an 1...
	sections := this.makeSections(str, span, width, len([]rune(separator)), rest...)
	// NOTE: we are skipping the last section as it already places the
	//		overflow symbol in the right spot...
	for i := 0; i < len(sections)-2; i += 2 {
		str, overflow := sections[i], sections[i+1]
		sep := separator
		if len(overflow) > 0 {
			if len(sep) == 0 {
				r := []rune(str)
				r[len(r)-1] = []rune(overflow)[0]
				str = string(r) 
			} else {
				sep = overflow } }
		sections[i], sections[i+1] = str, sep }
	// borders...
	if sections[len(sections)-1] == "" {
		sections[len(sections)-1] = border_r 
	// overflow + no borders -> place last overflow on last char...
	} else if border_r == "" {
		i := len(sections)-2
		// add space for the overflow char in the last non-eopty...
		// XXX is this correct -- we could remove a space from both the 
		//		section as well as from a separator...
		s := []rune(sections[i])
		for i > 0 && 
				len(s) == 0 {
			i--
			s = []rune(sections[i]) }
		// XXX handle escape sequences correctly...
		if len(s) > 0 {
			sections[i] = string(s[:len(s)-1]) } } 
	return append([]string{ border_l }, sections...) }

// XXX IGNORE_EMPTY
// XXX revise...
// XXX move...
// XXX cache...
func (this *Lines) Len() int {
	l := 0
	for _, r := range this.Lines {
		if r.Populated &&
				len(r.Text) > 0 {
			l++ } }
	return l }
func (this *Lines) PopulatedIndex(index int) int {
	for i, r := range this.Lines {
		if !r.Populated ||
				len(r.Text) == 0 {
			continue }
		if index == 0 {
			return i }
		index-- } 
	return -1 }

func (this *Lines) MakeEnv() Env {
	fill := " "
	if this.Filler != 0 {
		fill = string(this.Filler) }
	// positioning...
	l := this.Len()
	// XXX IGNORE_EMPTY
	i := this.Index
	//i := this.PopulatedIndex(this.Index)
	//i := this.RowOffset + this.Index
	// test and friends...
	var text, text_left, text_right string
	if i < l {
		//text = this.Lines[i].Text
		text = this.At(i).Text
		sections := this.splitSections(text, this.SpanMode != "none")
		text_left = sections[0]
		if len(sections) > 1 {
			text_right = sections[len(sections)-1] } }

	selection := this.LinesBuffer.Selected()
	selected := ""
	if l := len(selection); l > 0 {
		selected = fmt.Sprint(l) }

	env := Env {
		// used for '%F' placeholder value...
		"__F": fill,

		// XXX should this be a var or a placeholder???
		//"F": fill,
		"INDEX": fmt.Sprint(i),
		"LINE": fmt.Sprint(i + 1),
		"LINES": fmt.Sprint(l),
		"TEXT": text,
		"TEXT_LEFT": text_left,
		"TEXT_RIGHT": text_right,
		"SELECTION": strings.Join(selection, "\n"),
		"SELECTED": selected,
		// XXX TEST...
		"ACTIVE": strings.Join(this.Active(), "\n"),
	}
	for k, v := range this.Env {
		env[k] = v }

	return env }

// Template parsing...
//
//	Environment variables
//		$NAME
//		${NAME}
//		${NAME:- .. }
//		${NAME:+ .. }
//		${NAME:! .. }
//
//	Placeholders
//		These support the same syntax as the Environment variables with 
//		the "$" replaced with "%".
//
type AST struct {
	Type rune
	Head string
	Tail []AST
}
type Placeholders map[string] func(*Lines, Env) string
var PLACEHOLDERS = Placeholders {
	// XXX should this be a var or a placeholder???
	"F": func(this *Lines, env Env) string {
		fill, ok := env["__F"]
		if !ok {
			fill = " "
			if this.Filler != 0 {
				fill = string(this.Filler) } }
		return fill },
	"CMD": func(this *Lines, env Env) string {
		cmd, ok := env["CMD"]
		if ! ok {
			return "" }
		res := ""
		// XXX call the command...
		fmt.Println("XXX", cmd)
		return res },
	"S": func(this *Lines, env Env) string {
		return this.Spinner.String() },
}
type Expander func([]AST) string
type Expanders map[string]func(string, []AST, Expander) string
var EXPRESSION_HANDLERS = Expanders {
	// ${NAME?A:B}
	"?": func(value string, ast []AST, expand Expander) string {
		// split the ast at ":"...
		A := ast
		B := []AST{}
		for i, e := range ast {
			if e.Type == 0 {
				parts := strings.SplitN(e.Head, ":", 2)
				// we have a hit...
				if len(parts) > 1 {
					B = slices.Clone(ast[i:])
					A = ast[:i]
					A = append(A, AST{ Head: parts[0] })
					B[i].Head = parts[1]
					break } } }
		if value != "" {
			return expand(A) } 
		return expand(B) },
	// ${NAME:-DEFAULT}
	":-": func(value string, ast []AST, expand Expander) string {
		if value == "" {
			return expand(ast) }
		return value },
	// ${NAME:+ALTERNATIVE}
	":+": func(value string, ast []AST, expand Expander) string {
		if value != "" {
			return expand(ast) }
		return "" },
	// ${NAME:!IF_UNSET}
	":!": func(value string, ast []AST, expand Expander) string {
		if value == "" {
			return expand(ast) }
		return "" },
} 
var varPattern = regexp.MustCompile(
	// $$ / %%
	`([%$]{2}`+
		// $NAME / %NAME
		`|[$%][A-z_][A-z0-9_]*`+
		// ${NAME .. } / %{NAME .. }
		`|[$%]\{[A-z_][A-z0-9_]*`+
		// "\}"
		`|\\\}`+
		// "}"
		`|\}`+
		// normal text...
		`|[^$%}]*)`)
// XXX better error handling...
// XXX should we abstract out os.Getenv(..) ???
func (this *Lines) expandTemplate(str string, env Env) string {
	placeholders := PLACEHOLDERS 
	if this.Placeholders != nil {
		placeholders = *this.Placeholders }
	expressionHandlers := EXPRESSION_HANDLERS

	// Lex stage...
	//
	lex := []string{}
	varPattern.ReplaceAllStringFunc(
		str,
		func(match string) string {
			lex = append(lex, match) 
			return "" })

	// Parse stage...
	//
	var parseVar func(string, []string) (AST, []string)
	group := func(lex []string) ([]AST, []string) {
		res := []AST{}
		for len(lex) > 0 {
			cur := lex[0]
			lex = lex[1:]
			if cur == "" {
				continue }
			// special cases...
			if cur == "$$" || cur == "%%" {
				res = append(res, AST{ Head: string(cur[0]) })
				continue }
			switch cur[0] {
				case '}':
					// XXX GO: ^*&^#*$: can't use break here because in 
					//		Go a break at end of a case is automatic but
					//		it still is a switch keyword...
					return res, lex
				case '%', '$':
					var v AST
					v, lex = parseVar(cur, lex)
					res = append(res, v) 
				default:
					res = append(res, AST{ Head: cur }) } }
		return res, lex }
	parseVar = func(head string, lex []string) (AST, []string) {
		res := AST{
			Type: rune(head[0]),
		}
		if head[1] == '{' {
			res.Head = string(head[2:])
			res.Tail, lex = group(lex)
		} else {
			res.Head = string(head[1:]) }
		return res, lex }
	ast, rest := group(lex)
	if len(rest) > 0 {
		if rest[0] == "}" {
			log.Panicf("ExpandTemplate(..): Unexpected \"}\": %#v", str) }
		log.Panicf("ExpandTemplate(..): Unexpected end of input while parsing: %#v", str) }

	// Expand stage...
	//
	var expand Expander
	handleExpression := func(value string, ast []AST) string {
		if len(ast) == 0 {
			return value }
		next := ast[0].Head
		for prefix, f := range expressionHandlers {
			if prefix == string(next[:len(prefix)]) {
				ast[0].Head = string(next[len(prefix):])
				return f(value, ast, expand) } } 
		return value }
	expand = func(ast []AST) string {
		res := []string{}
		for _, a := range ast {
			// strings...
			if a.Type == 0 {
				res = append(res, a.Head, expand(a.Tail)) 
				continue }
			// $NAME / %NAME
			switch a.Type {
				case '$':
					value := ""
					if val, ok := env[a.Head] ; ok {
						value = val
					} else {
						value = os.Getenv(a.Head) }
					res = append(res, 
						handleExpression(
							value, 
							a.Tail))
				case '%':
					if f, ok := placeholders[a.Head]; ok {
						res = append(res, 
							handleExpression(
								f(this, env),
								a.Tail)) 
					} else {
						// XXX include .Tail???
						res = append(res, "%"+ a.Head) } } }
		return strings.Join(res, "") }

	return expand(ast) }

func (this *Lines) drawCells(col, row int, str string, style string) int {
	if this.CellsDrawer != nil {
		n, s := this.GetStyle(style)
		return this.CellsDrawer.drawCells(col, row, str, n, s)
	} else {
		fmt.Print(str) 
		// XXX account for ANSI escape sequences...
		return len([]rune(str)) } }
func (this *Lines) drawLine(col, row int, sections []string, style string) *Lines {
	overflow := string(OVERFLOW_INDICATOR)
	if len(this.OverflowIndicator) != 0 {
		overflow = string([]rune(this.OverflowIndicator)[0]) }

	// add offset...
	col += this.Left
	row += this.Top

	// draw...
	col += this.drawCells(col, row, sections[0], "border")
	i := 1
	for ; i < len(sections)-2; i+=2 {
		section, sep := sections[i], sections[i+1]
		col += this.drawCells(col, row, section, style +"-text")
		if sep == overflow {
			col += this.drawCells(col, row, sep, style +"-overflow")
		} else {
			col += this.drawCells(col, row, sep, style +"-separator") } }
	col += this.drawCells(col, row, sections[i], style +"-text")
	col += this.drawCells(col, row, sections[i+1], "border")
	// let the .drawCells(..) know the line is done...
	// NOTE: this is here to help the overloading code handle/ignore 
	//		newlines... 
	this.drawCells(col, row, "\n", "EOL")
	return this }

// NOTE: this adds $F to the env containing the current fill character.
// XXX move to .Lines.Iter()
func (this *Lines) Draw() *Lines {
	if this.Width <= 0 || this.Height <= 0 {
		return this }
	overflow := string(OVERFLOW_INDICATOR)
	if len(this.OverflowIndicator) != 0 {
		overflow = string([]rune(this.OverflowIndicator)[0]) }
	border := this.Border
	if len(border) > 0 && 
			len(border) < 8 {
		border += strings.Repeat(" ", 8 - len(border)) }
	rows := this.Rows()
	row := 0
	col := 0

	var env Env

	// title...
	corner_l := ""
	corner_r := ""
	border_h := " "
	top_line := 0
	if ! this.TitleDisabled || 
			border != "" {
		top_line = 1
		env = this.MakeEnv()
		if border != "" {
			corner_l = string([]rune(border)[1])
			corner_r = string([]rune(border)[3]) 
			border_h = string([]rune(border)[2]) }
		env["__F"] = border_h
		sections := this.makeSectionChrome(
			this.expandTemplate(this.Title, env), 
			this.SpanModeTitle,
			this.Width, 
			"", corner_l, corner_r, border_h)
		this.drawLine(col, row, sections, "title") 
		row++ }

	// content...
	//
	// border...
	border_l := ""
	border_r := ""
	if border != "" {
		border_l = string([]rune(border)[0])
		border_r = string([]rune(border)[4]) }
	// scrollbar...
	scrollbar := false
	var scrollbar_fg, scrollbar_bg string
	var scroller_size, scroller_offset int
	if ! this.ScrollbarDisabled {
		scrollbar_fg = string([]rune(SCROLLBAR)[0])
		scrollbar_bg = string([]rune(SCROLLBAR)[1])
		if this.Scrollbar != "" {
			scrollbar_fg = string([]rune(this.Scrollbar)[0])
			scrollbar_bg = string([]rune(this.Scrollbar)[1]) }
		l := this.Len()
		//if len(this.Lines) > rows {
		if l > rows {
			scrollbar = true 
			//r := float64(rows) / float64(len(this.Lines))
			r := float64(rows) / float64(l)
			scroller_size = 1 + int(float64(rows - 1) * r)
			scroller_offset = int(float64(this.RowOffset + 1) * r) } }
	// lines...
	sections := []string{}
	// XXX ITER
	// XXX add option to show empty...
	lines := IterStepper(this.Range(this.RowOffset))
	for i := 0; i < rows; i++ {
		text := ""
		// get line...
		selected := false
		line, ok := <-lines
		if ok {
			selected = line.Selected == true
			text = string([]rune(line.Text)[this.ColOffset:]) 
			// pre-style -- strip ansi escape codes from selected/current lines...
			if selected ||
					row == this.Index - this.RowOffset + top_line {
				text = string(StripANSIEscSeq([]rune(text))) }
		// no lines left -- generate template for empty lines...
		} else if ! this.SpanNoExtend {
			s := this.SpanMarker
			if s == "" {
				s = SPAN_MARKER }
			// no lines in buffer or last line was empty...
			if len(sections) == 0 {
				text = ""
			} else {
				n := int(float64(len(sections) - 3) / 2)
				// XXX we can directly generate a slice instead of parsing this...
				text = strings.Repeat(s, n) } }
		// line...
		if scrollbar || 
				! this.OverflowOverBorder {
			s := border_r
			if scrollbar {
				s = scrollbar_fg
				if scroller_offset > i || scroller_offset + scroller_size <= i {
					s = scrollbar_bg } }
			sections = this.makeSectionChrome(
				text, 
				// XXX should this be: this.SpanMode,
				"default",
				this.Width - len([]rune(border_l)) - len([]rune(s)),
				this.SpanSeparator, "", "")
			sections[0] = border_l
			// XXX can we get a case when the last section is neither "" nor overflow???
			//* XXX overflow as separator...
			if sections[len(sections)-1] == overflow {
				sections = append(sections, "", s)
			} else {
				sections[len(sections)-1] = s }
			/*/ // XXX overflow as part of text...
			if sections[len(sections)-1] == overflow {
				sections[len(sections)-2] += overflow }
			sections[len(sections)-1] = s
			//*/
		// line with overflow over border...
		} else {
			sections = this.makeSectionChrome(
				text, 
				// XXX should this be: this.SpanMode,
				"default",
				this.Width, 
				this.SpanSeparator, border_l, border_r) }
		// style...
		style := "normal"
		if row == this.Index - this.RowOffset + top_line {
			style = "current" } 
		if selected {
			if style == "current" {
				style = "current-selected"
			} else {
				style = "selected" } }
		this.drawLine(col, row, sections, style) 
		row++ }

	// status...
	if ! this.StatusDisabled {
		if len(env) == 0 {
			env = this.MakeEnv() }
		if border != "" {
			corner_l = string([]rune(border)[5])
			corner_r = string([]rune(border)[7])
			border_h = string([]rune(border)[6]) }
		env["__F"] = border_h
		sections := this.makeSectionChrome(
			this.expandTemplate(this.Status, env), 
			this.SpanModeStatus,
			this.Width, 
			"", corner_l, corner_r, border_h)
		this.drawLine(col, row, sections, "status") }

	return this }




/*
func main(){
	lines := Lines{}

	makeSectionChrome := func(s string, w int, r ...string) string {
		return strings.Join(lines.makeSectionChrome(s, "default", w, r...), "") }

	fmt.Println("")
	fmt.Println(">"+
		makeSectionChrome("1234567890", 10) + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20, "|") + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20, "|", "[", "]") + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20, "", "[", "]") + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20, "", "[", "]", "-") + "<")
	fmt.Println(">"+
		makeSectionChrome("", 20, "", "[", "]", "-") + "<")
	fmt.Println(">"+
		makeSectionChrome("", 20, []string{"", "┌", "┐", "─"}...) + "<")
	fmt.Println(">"+
		makeSectionChrome("A|B", 20, []string{"", "└", "┘", "─"}...) + "<")
	fmt.Println(">"+
		makeSectionChrome("|", 20, "", "[", "]", "-") + "<")
	fmt.Println(">>"+
		makeSectionChrome("moo|foo", 20-2, "", "", "", "-") + "<<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow overflow overflow", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20, "[[", "]]") + "<")
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20) + "<")
	//lines.SpanSeparator = "|"
	lines.SpanSeparator = "│"
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow|foo", 20) + "<")
	lines.SpanMode = "50%"
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow|foo", 20) + "<")
	lines.SpanMode = "*,*,*"
	fmt.Println(">"+
		makeSectionChrome("moo|foo|boo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("over|flow|over|flow", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("0123456789|0123456789|0123456789", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("under|flow", 20) + "<")
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	fmt.Println(">"+
		makeSectionChrome("", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("o|v|e|r|f|l|o|w", 20) + "<")


	testLineSizes := func(str string, r ...string){
		err := false
		printed := false
		for i := 4; i < 40; i++ {
			s := makeSectionChrome(str, i, r...)
			if len([]rune(s)) != i {
				err = true
				if ! printed {
					printed = true
					fmt.Println("ERR: \""+ str +"\"") }
				fmt.Println(">"+ s + "<") 
				fmt.Printf("^%"+ fmt.Sprint(i) +"v^\n"+
						"\tshould be: %v got: %v\n"+
						"\tsizes: %v\n", 
					"", i, len(s), 
					lines.parseSizes(lines.SpanMode, i, len([]rune(lines.SpanSeparator)))) } } 
		if ! err {
			fmt.Println("OK:", str) } }

	fmt.Println("")
	lines.SpanMode = ""
	testLineSizes("moo")
	testLineSizes("moo|foo")
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	testLineSizes("o|v|e|r|f|l|o|w")
	

	fmt.Println("")
	lines.SpanMode = ""
	fmt.Println(">"+
		makeSectionChrome("moo|foo", 20) +"<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow overflow overflow", 20) +"<")
	lines.Border = "│┌─┐│└─┘"
	fmt.Println(
		makeSectionChrome("moo|foo", 22))
	fmt.Println(
		makeSectionChrome("overflow overflow overflow overflow overflow overflow", 22))
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	fmt.Println(
		makeSectionChrome("o|v|e|r|f|l|o|w", 22))

	testSizes := func(s string, w int, p int){
		fmt.Println("w:", w, "sep:", p, "s: \""+ s +"\" ->", 
			lines.parseSizes(s, w, p)) }

	testBorderedSize := func(str string, w int){
		s := makeSectionChrome(str, w)
		fmt.Println("")
		fmt.Println(str)
		testSizes("*,*,*,*,*,*,*,*,*,*", w, 0)
		testSizes("*,*,*,*,*,*,*,*,*,*", w, 1)
		fmt.Printf("v%"+ fmt.Sprint(w) +"vv\n", "")
		fmt.Printf("%#v\n", s) 
		c := len(lines.Border)
		if c > 0 {
			c = len(string([]rune(lines.Border)[0])) + 
				len(string([]rune(lines.Border)[4])) }
		//if len([]rune(s)) - c != w {
		if len([]rune(s)) != w {
			//fmt.Println("length: expected:", w, "got:", len([]rune(s)))
			fmt.Printf("\tparseSizes: %#v\n",
				lines.parseSizes(lines.SpanMode, w-c, 1))
			fmt.Printf("\tmakeSections: %#v\n",
				lines.makeSections(str, "default", w-c, 1))
			fmt.Println("    -> ERR") 
		} else {
			fmt.Println("    -> OK") } }
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	lines.Border = "│┌─┐│└─┘"
	testBorderedSize("o|v|e|r|f|l|o|w", 20)
	testBorderedSize("o|v|e|r|f|l|o|w", 21)
	testBorderedSize("o|v|e|r|f|l|o|w", 22)
	testBorderedSize("o|v|e|r|f|l|o|w", 23)


	fmt.Println("")
	lines.SpanMode = "*,5"
	lines.Width = 20
	lines.Height = 6
	lines.Border = "│┌─┐│└─┘"
	lines.Write(
		"This|is\n"+
		"some text\n"+
		"\n"+
		"This is also\n"+
		"some|more text\n"+
		// XXX need to extend the separator from the above line...
		"\n"+
		"\n")
	lines.Draw()

	fmt.Println("")
	lines.Write("a single|line")
	lines.Draw()

	fmt.Println("")
	lines.Title = "A|B"
	lines.Status = "A|B"
	lines.Draw()
}
//*/

// vim:set ts=4 sw=4 :
