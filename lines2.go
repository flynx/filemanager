
package main

import (
	"fmt"
	"log"
	"io"
	"strings"
	"strconv"
	//"slices"
	"bufio"
	"sync"
	"regexp"
	"os"
)


// XXX is this too generic???
type Env map[string]string


// Row
//
type Row struct {
	selected bool
	transformed bool
	populated bool
	text string
}



// LinesBuffer
//
type LinesBuffer struct {
	sync.Mutex
	Lines []Row
	Width int
}

func (this *LinesBuffer) Clear() *LinesBuffer {
	this.Lines = []Row{}
	this.Width = 0
	return this }
func (this *LinesBuffer) String() string {
	lines := []string{}
	for _, line := range this.Lines {
		lines = append(lines, line.text) }
	return strings.Join(lines, "\n") }
func (this *LinesBuffer) Push(line string) *LinesBuffer {
	this.Lines = append(this.Lines, Row{ text: line })
	l := len([]rune(line))
	if this.Width < l {
		this.Width = l }
	return this }
func (this *LinesBuffer) Append(in any) *LinesBuffer {
	switch in.(type) {
		// XXX this is covered by default, do we need this case???
		//case string:
		//	for _, str := range strings.Split(in.(string), "\n") {
		//		this.Push(str) }
		case io.Reader:
			scanner := bufio.NewScanner(in.(io.Reader))
			for scanner.Scan(){
				this.Push(scanner.Text()) } 
		default:
			for _, str := range strings.Split(fmt.Sprint(in), "\n") {
				this.Push(str) } }
	return this }
func (this *LinesBuffer) Write(in any) *LinesBuffer {
	//this.Lock()
	//defer this.Unlock()
	return this.
		Clear().
		Append(in) }



// Placeholders
//
type Placeholders map[string] func(*Lines, Env) string
var PLACEHOLDERS = Placeholders {
	"CMD": func(this *Lines, env Env) string {
		cmd, ok := env["CMD"]
		if ! ok {
			return "" }
		res := ""
		// XXX call the command...
		fmt.Println("---", cmd)
		return res },
}



// CellsDrawer
//
// XXX not sure how to define an easily overloadable/extendable "object"... 
//		...don't tell me that a Go-y solution is passing function pointers))))
// XXX revise name... 
type CellsDrawer interface {
	drawCells(col, row int, str string, style string)
}



// Lines
//

var TAB_SIZE = 8

var OVERFLOW_INDICATOR = '}'

var SPAN_MARKER = "%SPAN"
// XXX this would require us to support escaping...
//var SPAN_MARKER = "|"
var SPAN_MIN_WIDTH = 5

//var SCROLLBAR = "█░"
var SCROLLBAR = "┃│"

// XXX should this be Reader/Writer???
type Lines struct {
	CellsDrawer

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
	CurrentRow int

	Env Env

	//Theme Theme

	// chrome...
	Title string
	HideTitle bool
	Status string
	HideStatus bool

	TabSize int

	OverflowIndicator rune

	// Format: 
	//		"│┌─┐│└─┘"
	//		 01234567
	Border string
	OverflowOverBorder bool

	// Format: 
	//		"█░"
	//		 01
	// NOTE: if this is set to "" the default SCROLLBAR will be used.
	Scrollbar string
	ScrollbarDisabled bool

	Filler rune

	// column spanning...
	SpanMode string
	// cache...
	__SpanMode struct {
		text string
		width int
		value []int
	}
	SpanMarker string
	SpanSeparator string
	SpanMinSize int
	SpanNoExtend bool

}
// XXX can we integrate this transparently???
var LinesDefaults = Lines {
	Title: "",
	Status: "%CMD%SPAN $LINE/$LINES ",
}

// XXX add support for escape sequences...
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
	//keep_non_printable := false

	// offset from i to printed rune index in runes...
	offset := 0
	// number of blanks to print from current position...
	// NOTE: to draw N blanks:
	//		- set blanks to N
	//		- either
	//			- decrement i -- will draw a blank on curent position... 
	//			- continue
	//		- or:
	//			- skip to this.drawCell(..)
	blanks := 0
	// NOTE: this can legally get longer than width if it contains escape 
	//		sequeces...
	output := []rune(strings.Repeat(" ", width))
	for i := 0 ; i < width; i++ {
		r := fill
		// blanks...
		if blanks > 0 {
			blanks--
			offset--
		// runes...
		} else {
			// get the rune...
			if i + offset < len(runes) {
				r = runes[i + offset] }
			/* XXX
			// escape sequences...
			// XXX add option to keep/remove escape sequences...
			if r == '\x1B' {
				seq := []rune{}

				// XXX

				// XXX if width was 0 and we are not including these we need 
				//		to truncate the output...
				if expand && keep_non_printable {
					output = append(output, make([]rune, len(seq))...) 
					width += len(seq)
				} else {
					output = output[:len(output)-len(seq)] 
					width -= len(seq) } }
			//*/
			// tab -- offset output to next tabstop... 
			if r == '\t' { 
				blanks = tab - (i % tab) - 1 
				continue } }
		// set the rune...
		output[i] = r }
	return string(output), 
		// overflow...
		len(runes) - offset > width }
func (this *Lines) parseSizes(str string, width int, sep int) []int {
	str = strings.TrimSpace(str)
	min_size := this.SpanMinSize
	if min_size == 0 {
		min_size = SPAN_MIN_WIDTH }
	spec := []string{}

	// special case: single col...
	if str == "100%" || 
			str == "*" ||
			str == "" {
		spec = []string{str}
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
			//if i < len(spec)-1 {
			//	rest -= sep }
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
			if i < len(spec)-1 {
				rest -= sep } }
		// min width...
		if cols > 0 && 
				cols < min_size {
			cols = min_size }
		if cols > 0 {
		rest -= cols }
		sizes = append(sizes, cols) }

	// fill *'s and trim overflow...
	star_size := 0
	if stars > 0 {
		star_size = int(float64(rest) / float64(stars) + 0.5)
		//fmt.Println("###", rest, "/", stars, "->", size, over)
		//if star_size != 0 && 
		//		star_size < min_size {
		if star_size < min_size {
			star_size = min_size } }
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
		// star...
		if size < 0 {
			size = star_size
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
func (this *Lines) makeSections(str string, width int, sep_size int, rest ...string) []string {
	// defaults...
	marker := this.SpanMarker
	if marker == "" {
		marker = SPAN_MARKER }
	overflow := string(OVERFLOW_INDICATOR)
	if this.OverflowIndicator != 0 {
		overflow = string(this.OverflowIndicator) }

	sections := strings.Split(str, marker)

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
		if this.SpanMode == "" || this.SpanMode == "fit-right" {
			section := doSection(sections[len(sections)-1], 0)
			l := len(section[0])
			sizes = this.parseSizes("*,"+ fmt.Sprint(l), width, sep_size)
		// sizing: manual (cached)...
		} else {
			// cached result -- the same for each line, no need to recalculate...
			if this.SpanMode == this.__SpanMode.text && 
					this.__SpanMode.width == width {
				sizes = this.__SpanMode.value
			// generate/cache...
			} else {
				sizes = this.parseSizes(this.SpanMode, width, sep_size)
				this.__SpanMode.text = this.SpanMode
				this.__SpanMode.width = width
				this.__SpanMode.value = sizes } } 
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
	return res }
//
//	.makeSectionChrome(<str>, <width>[<span_separator>[, <left_border>, <right_border>[, <filler>]]])
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
// XXX shoud we be able to distinguish between last cell overflow and 
//		and section overflow???
//		...currently it is not possible to do so...
// XXX make sure to handle/trim lines ending in escape sequences correctly 
//		when embedding overflow indicator...
func (this *Lines) makeSectionChrome(str string, width int, rest ...string) []string {
	separator := this.SpanSeparator
	if len(rest) >= 1 {
		separator = rest[0]
		rest = rest[1:] }
	border_l := ""
	border_r := ""
	if len(rest) >= 2 {
		border_l = rest[0] 
		border_r = rest[1] 
		width -= len([]rune(border_l)) + len([]rune(border_r))
		rest = rest[2:]
	} else if this.Border != "" {
		border_l = string([]rune(this.Border)[0])
		border_r = string([]rune(this.Border)[4])
		width -= 2 }
	sections := this.makeSections(str, width, len([]rune(separator)), rest...)
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

func (this *Lines) makeEnv() Env {
	l := len(this.Lines)
	i := this.RowOffset + this.CurrentRow
	// test and friends...
	var text, text_left, text_right string
	if i < l {
		text = this.Lines[i].text
		marker := this.SpanMarker
		if marker == "" {
			marker = SPAN_MARKER }
		// XXX might be a good idea to support basic arrays.. (???)
		sections := strings.Split(text, marker)
		text_left = sections[0]
		if len(sections) > 1 {
			text_right = sections[len(sections)-1] } }

	env := Env {
		"INDEX": fmt.Sprint(i),
		"LINE": fmt.Sprint(i + 1),
		"LINES": fmt.Sprint(l),
		"TEXT": text,
		"TEXT_LEFT": text_left,
		"TEXT_RIGHT": text_right,
		// XXX ACTIVE -- selection or current...
		// XXX SELECTION
		// XXX SELECTED
	}
	for k, v := range this.Env {
		env[k] = v }

	return env }

// XXX add %CMD support...
var isTemplatePattern = regexp.MustCompile(`([%$]{2}|[$%][a-zA-Z_]+|[$%]\{[a-zA-Z_]+\})`)
func (this *Lines) expandTemplate(str string, env Env) string {
	// handle placeholders...
	marker := this.SpanMarker
	if marker == "" {
		marker = SPAN_MARKER }
	placeholders := PLACEHOLDERS 
	if this.Placeholders != nil {
		placeholders = *this.Placeholders }
	str = string(isTemplatePattern.ReplaceAllFunc(
		[]byte(str), 
		func(match []byte) []byte {
			// normalize...
			name := string(match[1:])
			// $NAME
			if match[0] == "$"[0] {
				if name[0] == '{' {
					name = string(name[1:len(name)-1]) }
				if name == "$" {
					return []byte(name) }
				// get the value...
				if val, ok := env[name] ; ok {
					return []byte(val)
				} else {
					return []byte(os.Getenv(name)) }
				return []byte{}
			// %NAME
			} else if match[0] == "%"[0] {
				if name[0] == '{' {
					name = string(name[1:len(name)-1]) }
				if name == "%" {
					return []byte(name) }
				if f, ok := placeholders[name]; ok {
					return []byte(f(this, env)) }
				// XXX should undefined placeholders get returned as-is (current) 
				//		or be blank???
				//return []byte{} }))
				return match }
			return match }))
	return str }

// XXX return/handle errors???
func (this *Lines) drawCells(col, row int, str string, style string) {
	if this.CellsDrawer != nil {
		this.CellsDrawer.drawCells(col, row, str, style)
	} else {
		fmt.Print(str) } }
func (this *Lines) drawLine(col, row int, sections []string, style string) *Lines {
	/*/ XXX STUB...
	fmt.Println(
		strings.Join(sections, ""))
	return this
	//*/
	this.drawCells(col, row, sections[0], "border")
	col += len(sections[0])
	i := 1
	for ; i < len(sections)-2; i+=2 {
		section, sep := sections[i], sections[i+1]
		this.drawCells(col, row, section, style +"-text")
		col += len(sections[i])
		this.drawCells(col, row, sep, style +"-separator") 
		col += len(sections[i]) }
	this.drawCells(col, row, sections[i], style +"-text")
	col += len(sections[i])
	this.drawCells(col, row, sections[i+1], "border")
	col += len(sections[i+1])
	// XXX STUB...
	fmt.Print("\n")
	return this }

// XXX sould be nice to control how we output from this level...
//		...i.e. Draw to string or draw to terminal...
func (this *Lines) Draw() *Lines {
	rows := this.Height
	if ! this.HideTitle {
		rows-- }
	if ! this.HideStatus {
		rows-- }
	row := 0
	col := 0

	var env Env

	// title...
	corner_l := ""
	corner_r := ""
	border_h := " "
	if ! this.HideTitle {
		env = this.makeEnv()
		if this.Border != "" {
			corner_l = string([]rune(this.Border)[1])
			corner_r = string([]rune(this.Border)[3]) 
			border_h = string([]rune(this.Border)[2]) }
		sections := this.makeSectionChrome(
			this.expandTemplate(this.Title, env), 
			this.Width, 
			"", corner_l, corner_r, border_h)
		this.drawLine(col, row, sections, "title") 
		row++ }

	// content...
	//
	// border...
	border_l := ""
	border_r := ""
	if this.Border != "" {
		border_l = string([]rune(this.Border)[0])
		border_r = string([]rune(this.Border)[4]) }
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
		if len(this.Lines) > rows {
			scrollbar = true 
			r := float64(rows) / float64(len(this.Lines))
			scroller_size = 1 + int(float64(rows - 1) * r)
			scroller_offset = int(float64(this.RowOffset + 1) * r) } }
	// lines...
	sections := []string{}
	for i := 0; i < rows; i++ {
		row := i + this.RowOffset
		text := ""
		var line Row
		// get line...
		if row < len(this.Lines) {
			line = this.Lines[row]
			text = string([]rune(line.text)[this.ColOffset:]) 
		// no lines left...
		} else if ! this.SpanNoExtend {
			s := this.SpanMarker
			if s == "" {
				s = SPAN_MARKER }
			n := int(float64(len(sections) - 3) / 2)
			// XXX we can directly generate a slice instead of parsing this...
			text = strings.Repeat(s, n) }
		// line...
		if scrollbar || 
				! this.OverflowOverBorder {
			s := border_r
			if scrollbar {
				s = scrollbar_fg
				if scroller_offset > i || scroller_offset + scroller_size <= i {
					s = scrollbar_bg } }
			sections = append(
				append([]string{border_l}, 
					this.makeSectionChrome(
						text, 
						this.Width - len([]rune(border_l)) - len([]rune(s)),
						this.SpanSeparator, "", "")...),
				s)
		// line with overflow over border...
		} else {
			sections = this.makeSectionChrome(
				text, 
				this.Width, 
				this.SpanSeparator, border_l, border_r) }
		// style...
		style := "normal"
		if row == this.CurrentRow {
			style = "current" }
		if line.selected {
			if style == "current" {
				style = "current-selected"
			} else {
				style = "selected" } }
		this.drawLine(col, row, sections, style) 
		row++ }

	// status...
	if ! this.HideStatus {
		if len(env) == 0 {
			env = this.makeEnv() }
		if this.Border != "" {
			corner_l = string([]rune(this.Border)[5])
			corner_r = string([]rune(this.Border)[7])
			border_h = string([]rune(this.Border)[6]) }
		sections := this.makeSectionChrome(
			this.expandTemplate(this.Status, env), 
			this.Width, 
			"", corner_l, corner_r, border_h)
		this.drawLine(col, row, sections, "status") }

	return this }





func main(){
	lines := Lines{}

	PLACEHOLDERS["TEST"] = 
		func(this *Lines, env Env) string {
			v, ok := env["TEST"]
			if ! ok {
				env["TEST"] = "1"
			} else {
				if i, err := strconv.Atoi(v); err == nil {
					env["TEST"] = fmt.Sprint(i+1) } }
			return "test string " + env["TEST"] }
	env := lines.makeEnv()
	fmt.Println(lines.expandTemplate(`
Template expansion test:
	$$MOO: $MOO
	$$INDEX: $INDEX
	$$LINE: $LINE
	$$LINES: $LINES
	%%MOO: %MOO
	%%%%: %%
	%%TEST: %TEST
	%%TEST: %TEST
	$$TEST: $TEST
	%%CMD: %CMD
	`, env))

	makeSection := func(s string, w int) string {
		s, o := lines.makeSection(s, w)
		if o {
			r := []rune(s)
			r[len(r)-1] = '}' 
			s = string(r) }
		return s }

	fmt.Println("")
	fmt.Println(">"+ makeSection("no overflow", 0) +"<")
	fmt.Println(">"+ makeSection("no overflow no overflow no overflow no overflow", 0) +"<")
	fmt.Println(">"+ makeSection("a b c", 20) +"<")
	fmt.Println(">"+ makeSection("tab     b       c", 20) +"<")
	fmt.Println(">"+ makeSection("tab\tb\tc", 20) +"<")
	fmt.Println(">"+ makeSection("overflow overflow overflow overflow overflow", 20) +"<")
	fmt.Println(">"+ makeSection("tab overflow\t\t\t\tmoo", 20) +"<")


	makeSectionChrome := func(s string, w int, r ...string) string {
		return strings.Join(lines.makeSectionChrome(s, w, r...), "") }

	fmt.Println("")
	fmt.Println(">"+
		makeSectionChrome("1234567890", 10) + "<")
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow overflow overflow", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo", 20, "[[", "]]") + "<")
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo", 20) + "<")
	//lines.SpanSeparator = "|"
	lines.SpanSeparator = "│"
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow%SPANfoo", 20) + "<")
	lines.SpanMode = "50%"
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow%SPANfoo", 20) + "<")
	lines.SpanMode = "*,*,*"
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo%SPANboo", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("over%SPANflow%SPANover%SPANflow", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("0123456789%SPAN0123456789%SPAN0123456789", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("under%SPANflow", 20) + "<")
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	fmt.Println(">"+
		makeSectionChrome("", 20) + "<")
	fmt.Println(">"+
		makeSectionChrome("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 20) + "<")


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
	testLineSizes("moo%SPANfoo")
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	testLineSizes("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw")
	

	fmt.Println("")
	lines.SpanMode = ""
	fmt.Println(">"+
		makeSectionChrome("moo%SPANfoo", 20) +"<")
	fmt.Println(">"+
		makeSectionChrome("overflow overflow overflow overflow overflow overflow", 20) +"<")
	lines.Border = "│┌─┐│└─┘"
	fmt.Println(
		makeSectionChrome("moo%SPANfoo", 22))
	fmt.Println(
		makeSectionChrome("overflow overflow overflow overflow overflow overflow", 22))
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	fmt.Println(
		makeSectionChrome("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 22))

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
				lines.makeSections(str, w-c, 1))
			fmt.Println("    -> ERR") 
		} else {
			fmt.Println("    -> OK") } }
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	lines.Border = "│┌─┐│└─┘"
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 20)
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 21)
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 22)
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 23)


	fmt.Println("")
	lines.SpanMode = "*,5"
	lines.Width = 20
	lines.Height = 6
	lines.Border = "│┌─┐│└─┘"
	lines.Write(
		"This%SPANis\n"+
		"some text\n"+
		"\n"+
		"This is also\n"+
		"some%SPANmore text\n"+
		// XXX need to extend the separator from the above line...
		"\n"+
		"\n")
	lines.Draw()

	fmt.Println("")
	lines.Write("a single%SPANline")
	lines.Draw()
}

// vim:set ts=4 sw=4 :
