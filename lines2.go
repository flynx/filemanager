
package main

import "fmt"
import "io"
import "strings"
import "bufio"
import "sync"


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
	this.Lock()
	defer this.Unlock()
	return this.
		Clear().
		Append(in) }

// Liner
//
// XXX not sure how to define an easily overloadable/extendable "object"... 
//		...don't tell me that a Go-y solution is passing function pointers))))
// XXX revise name... 
type Liner interface {
	// XXX need to cast style to an apporpriate type in the implementation...
	drawCell(col, row int, r rune, style any) *Liner
}

var TAB_SIZE = 8

var OVERFLOW_INDICATOR = '}'

var SPAN_MARKER = "%SPAN"

// Lines
//
// XXX should this be Reader/Writer???
type Lines struct {
	Liner

	// XXX is this a good idea???
	*LinesBuffer

	Top int
	Left int
	Width int
	Height int
	Border int

	//Theme Theme

	Title string
	Status string

	TextOffsetV int
	TextOffsetH int

	TabSize int

	OverflowIndicator rune

	SpanMode string
	SpanMarker string
	SpanSeparator rune

}
func (this *Lines) makeSection(str string, width int) (string, bool) {
	// defaults...
	tab := this.TabSize
	if tab == 0 {
		tab = TAB_SIZE }
	fill := ' '

	runes := []rune(str)
	expand := true
	if width == 0 {
		expand = false
		width = len(runes) }
	keep_non_printable := false

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
func (this *Lines) makeSections(str string, width int) []string {
	marker := this.SpanMarker
	if marker == "" {
		marker = SPAN_MARKER }
	//sep := this.SpanSeparator
	overflow := this.OverflowIndicator

	sections := strings.Split(str, marker)

	doSection := func(str string, width int) []string {
		str, o := this.makeSection(sections[0], width)
		sep := ""
		if o {
			sep = string(overflow) }
		return []string{ 
			str, 
			sep, 
		} }


	// single section...
	if len(sections) == 1 {
		return doSection(sections[0], width)

	} else {
		l := []string{}
		for _, section := range sections {
			l = append(l, doSection(section, 0)...) }
		// automatic...
		if this.SpanMode == "" || this.SpanMode == "fit-right" {
			

		// manual...
		} else {
		}
	}

	return sections }
// XXX
func (this *Lines) expandTemplate(tpl string) string {
	// XXX
	return tpl }




func main(){
	lines := Lines{}

	withOverflow := func(s string, w int) string {
		s, o := lines.makeSection(s, w)
		if o {
			r := []rune(s)
			r[len(r)-1] = '}' 
			s = string(r) }
		return s }

	fmt.Println(">"+ withOverflow("no overflow", 0) +"<")
	fmt.Println(">"+ withOverflow("no overflow no overflow no overflow no overflow", 0) +"<")
	fmt.Println(">"+ withOverflow("a b c", 20) +"<")
	fmt.Println(">"+ withOverflow("tab     b       c", 20) +"<")
	fmt.Println(">"+ withOverflow("tab\tb\tc", 20) +"<")
	fmt.Println(">"+ withOverflow("overflow overflow overflow overflow overflow", 20) +"<")
	fmt.Println(">"+ withOverflow("tab overflow\t\t\t\tmoo", 20) +"<")
}


