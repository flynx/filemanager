
package main

import "fmt"
import "log"
import "io"
import "strings"
import "strconv"
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
// XXX this would require us to support escaping...
//var SPAN_MARKER = "|"
var SPAN_MIN_WIDTH = 5

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

	//Theme Theme

	Title string
	Status string

	TextOffsetV int
	TextOffsetH int

	TabSize int

	OverflowIndicator rune

	SpanMode string
	SpanMarker string
	SpanSeparator string
	SpanMinSize int

	Border string
}
var LinesDefaults = Lines {
	Title: "",
	Status: "%CMD%SPAN $LINE/$LINES ",
}
// XXX add support for escape sequences...
func (this *Lines) makeSection(str string, width int) (string, bool) {
	// defaults...
	tab := this.TabSize
	if tab == 0 {
		tab = TAB_SIZE }
	fill := ' '

	runes := []rune(str)
	//expand := true
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
func (this *Lines) parseSizes(str string, width int) []int {
	min_size := this.SpanMinSize
	if min_size == 0 {
		min_size = SPAN_MIN_WIDTH }
	sizes := []int{}
	stars := []int{}
	rest := width
	// XXX can we pre-parse this once???
	for i, size := range strings.Split(str, ",") {
		size = strings.TrimSpace(size)
		cols := 0
		if size == "*" || 
				size == "" {
			stars = append(stars, i)
		} else if size[len(size)-1] == '%' {
			p, err := strconv.ParseFloat(string(size[:len(size)-1]), 64)
			if err != nil {
				log.Println("Error parsing:", size, "in:", str) 
				stars = append(stars, i)
				continue }
			cols = int(float64(width) * (p / 100))
		} else {
			var err error
			cols, err = strconv.Atoi(size) 
			if err != nil {
				log.Println("Error parsing:", size, "in:", str) 
				stars = append(stars, i)
				continue } }
		if cols != 0 && 
				cols < min_size {
			cols = min_size }
		rest -= cols
		sizes = append(sizes, cols) }
	// fill "*"'s
	if len(stars) > 0 {
		r := int(float64(rest) / float64(len(stars)))
		if r != 0 && 
				r < min_size {
			r = min_size }
		rest = rest % len(stars)
		i := 0
		for _, i = range stars {
			c := 0
			// spread the overflow between cells...
			if rest > 0 {
				rest--
				c = 1 }
			sizes[i] = r + c }
		//sizes[i] += rest
	// add the rest of the cols to one last column...
	} else {
		sizes = append(sizes, rest) }
	return sizes }
func (this *Lines) makeSections(str string, width int, sep_size int) []string {
	marker := this.SpanMarker
	if marker == "" {
		marker = SPAN_MARKER }
	overflow := string(OVERFLOW_INDICATOR)
	if this.OverflowIndicator != 0 {
		overflow = string(this.OverflowIndicator) }

	sections := strings.Split(str, marker)

	skip := false
	doSection := func(str string, width int, sep_size int) []string {
		str, o := this.makeSection(str, width - sep_size)
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
		return doSection(sections[0], width, 0)

	} else {
		// sizing: automatic...
		sizes := []int{}
		if this.SpanMode == "" || this.SpanMode == "fit-right" {
			// XXX avoid reprocessing this section below (???)
			section := doSection(sections[len(sections)-1], 0, 0)
			l := len(section[0])
			sizes = this.parseSizes(fmt.Sprint("*,", l), width)
		// sizing: manual...
		} else {
			sizes = this.parseSizes(this.SpanMode, width) }
		// build the sections...
		var i int
		getSection := func(i int) string {
			section := ""
			if i < len(sections) {
				section = sections[i] }
			return section }
		rest := width
		for i=0; i < len(sizes)-1; i++ {
			// do not process stuff that will get off screen...
			if rest <= 0 {
				skip = true
				sizes[i] += rest - sep_size
				break }
			rest -= sizes[i] + sep_size
			res = append(res, doSection(getSection(i), sizes[i], sep_size)...) } 
		// last section...
		if sizes[i] == 0 {
			res = append(res, "", overflow)
		} else if sizes[i] > 0 {
			res = append(res, doSection(getSection(i), sizes[i], 0)...) } }
	return res }
//
// Format:
//		<sections> ::= 
//			<empty>
//			| <section>
//			| <section> [ <res> ]
//		<section> ::= 
//			<text> <sep>
//
// NOTE: we are not joining the list here so as to enable further 
//		processing (e.g. styling) down the line...
// XXX shoud we be able to distinguish between last cell overflow and 
//		and section overflow???
//		...currently it is not possible to do so...
// XXX this only merges overflow indicator into sections, do we really 
//		need this or should we merge this into .makeSections(..)???
// XXX make sure to handle lines ending in escape sequences correctly 
//		when embedding overflow indicator...
func (this *Lines) makeNormSections(str string, width int) []string {
	separator := this.SpanSeparator
	sections := this.makeSections(str, width, len(separator))
	// NOTE: we are skipping the last section...
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
	return sections }

//func (this *Lines) makeTitleLine(str string, width int) []string {
//}
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
func (this *Lines) makeContentLine(str string, width int) []string {
	l := ""
	r := ""
	if this.Border != "" {
		l = string([]rune(this.Border)[0])
		r = string([]rune(this.Border)[4])
		width -= 2 }
	sections := this.makeNormSections(str, width)
	if sections[len(sections)-1] == "" {
		sections[len(sections)-1] = r 
	// overflow + no borders -> place last overflow on last char...
	} else if this.Border == "" {
		s := []rune(sections[len(sections)-2])
		// XXX handle escape sequences correctly...
		sections[len(sections)-2] = string(s[:len(s)-1]) }
	return append([]string{ l }, sections...) }
//func (this *Lines) makeStatusLine(str string, width int) []string {
//}


// XXX
func (this *Lines) expandTemplate(tpl string) string {
	// XXX
	return tpl }




func main(){
	lines := Lines{}


	testSizes := func(s string, w int){
		fmt.Println("SIZE PARSING: w:", w, "s: \""+ s +"\" ->", lines.parseSizes(s, w)) }

	testSizes("50%", 100)
	testSizes("50%", 101)
	testSizes("50%,", 101)
	testSizes("10,50%,10", 101)
	testSizes("10,*,10", 101)
	testSizes("10,*,*,10", 101)
	testSizes("*,*,*", 100)
	testSizes("*,*,*", 20)


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


	makeNormSections := func(s string, w int) string {
		l := lines.makeNormSections(s, w)
		s = strings.Join(l[:len(l)-1], "")
		if l[len(l)-1] == "}" {
			r := []rune(s)
			r[len(r)-1] = '}' 
			s = string(r) }
		return s }

	fmt.Println("")
	fmt.Println(">"+
		makeNormSections("moo%SPANfoo", 20) + "<")
	fmt.Println(">"+
		makeNormSections("overflow overflow overflow overflow overflow overflow", 20) + "<")
	lines.SpanSeparator = "|"
	fmt.Println(">"+
		makeNormSections("moo%SPANfoo", 20) + "<")
	fmt.Println(">"+
		makeNormSections("overflow overflow overflow overflow%SPANfoo", 20) + "<")
	lines.SpanMode = "50%"
	fmt.Println(">"+
		makeNormSections("moo%SPANfoo", 20) + "<")
	fmt.Println(">"+
		makeNormSections("overflow overflow overflow overflow%SPANfoo", 20) + "<")
	lines.SpanMode = "*,*,*"
	fmt.Println(">"+
		makeNormSections("moo%SPANfoo%SPANboo", 20) + "<")
	fmt.Println(">"+
		makeNormSections("over%SPANflow%SPANover%SPANflow", 20) + "<")
	fmt.Println(">"+
		makeNormSections("0123456789%SPAN0123456789%SPAN0123456789", 20) + "<")
	fmt.Println(">"+
		makeNormSections("under%SPANflow", 20) + "<")
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	fmt.Println(">"+
		makeNormSections("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 20) + "<")
	// XXX BUG this still is 20 wide...
	fmt.Println(">"+
		makeNormSections("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 18) + "<")
	// XXX BUG this still is more than 20 wide...
	fmt.Println(">"+
		makeNormSections("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 19) + "<")
	

	makeContentLine := func(s string, w int) string {
		return strings.Join(lines.makeContentLine(s, w), "") }

	fmt.Println("")
	lines.SpanMode = ""
	fmt.Println(">"+
		makeContentLine("moo%SPANfoo", 20) +"<")
	fmt.Println(">"+
		makeContentLine("overflow overflow overflow overflow overflow overflow", 20) +"<")
	lines.Border = "│┌─┐│└─┘"
	fmt.Println(
		makeContentLine("moo%SPANfoo", 20))
	fmt.Println(
		makeContentLine("overflow overflow overflow overflow overflow overflow", 20))
	lines.SpanMode = "*,*,*,*,*,*,*,*,*,*"
	fmt.Println(
		makeContentLine("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 20))
}


