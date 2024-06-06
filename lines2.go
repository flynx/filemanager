
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
	// cache...
	_SpanMode string
	_SpanMode_cache []int
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
// XXX is this overcomplicated???
func (this *Lines) parseSizes(str string, width int, sep int) []int {
	str = strings.TrimSpace(str)
	min_size := this.SpanMinSize
	if min_size == 0 {
		min_size = SPAN_MIN_WIDTH }
	spec := []string{}
	// special case...
	if str == "100%" {
		spec = []string{str}
	} else {
		spec = strings.Split(str, ",") 
		// only one col specified -> append "*"...
		if len(spec) < 2 {
			spec = append(spec, "*") } }
	rest := width
	sizes := []int{}
	stars := 0
	// build list of sizes...
	star_seps := 0
	for i, size := range spec {
		size = strings.TrimSpace(size)
		cols := -1
		// *...
		if size == "*" || 
				size == "" {
			stars++
			if i < len(spec)-1 {
				star_seps += sep }
		// %...
		} else if size[len(size)-1] == '%' {
			p, err := strconv.ParseFloat(string(size[:len(size)-1]), 64)
			if err != nil {
				log.Println("Error parsing:", size, "in:", str) 
				stars++
				sizes = append(sizes, cols)
				continue }
			cols = int(float64(width) * (p / 100))
			// accout for separators...
			if i < len(spec)-1 {
				cols -= sep }
		// cols (explicit)
		// NOTE: these do not include separators...
		} else {
			var err error
			cols, err = strconv.Atoi(size) 
			if err != nil {
				log.Println("Error parsing:", size, "in:", str) 
				stars++
				sizes = append(sizes, cols)
				continue } }
		if cols > 0 && 
				cols < min_size {
			cols = min_size }
		rest -= cols
		sizes = append(sizes, cols) }
	// fill *'s and trim overflow...
	star_size := 0
	if stars > 0 {
		star_size = int(float64(rest) / float64(stars))
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
func (this *Lines) makeSections(str string, width int, sep_size int) []string {
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
		str, o := this.makeSection(str, width)
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
			if this.SpanMode == this._SpanMode {
				sizes = this._SpanMode_cache 
			// generate/cache...
			} else {
				sizes = this.parseSizes(this.SpanMode, width, sep_size)
				this._SpanMode = this._SpanMode
				this._SpanMode_cache = sizes } }
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
//	.makeSectionChrome(<str>, <width>[, <left_border>[, <right_border>]])
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
// XXX this only merges overflow indicator and span separator into 
//		sections, do we really need this or should we merge this into 
//		.makeSections(..)???
// XXX make sure to handle lines ending in escape sequences correctly 
//		when embedding overflow indicator...
func (this *Lines) makeSectionChrome(str string, width int, rest ...string) []string {
	l := ""
	r := ""
	if len(rest) >= 2 {
		l = rest[0] 
		r = rest[1] 
		width -= len(rest[0]) + len(rest[1])
	} else if this.Border != "" {
		l = string([]rune(this.Border)[0])
		r = string([]rune(this.Border)[4])
		width -= 2 }
	separator := this.SpanSeparator
	sections := this.makeSections(str, width, len(separator))
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
		sections[len(sections)-1] = r 
	// overflow + no borders -> place last overflow on last char...
	} else if this.Border == "" {
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
	return append([]string{ l }, sections...) }

//func (this *Lines) makeTitleLine(str string, width int) []string {
//}
//func (this *Lines) makeStatusLine(str string, width int) []string {
//}

func (this *Lines) drawCell(r rune) *Lines {
	return this }
func (this *Lines) Draw(str string) *Lines {
	return this }

// XXX
func (this *Lines) expandTemplate(tpl string) string {
	// XXX
	return tpl }




func main(){
	lines := Lines{}


	testSizes := func(s string, w int, p int){
		fmt.Println("SIZE PARSING: w:", w, "sep:", p, "s: \""+ s +"\" ->", 
			lines.parseSizes(s, w, p)) }

	testSizes("50%", 100, 0)
	testSizes("50%", 101, 0)
	// XXX this yields an odd split of 49/51 -- can we make this more natural???
	testSizes("50%", 101, 1)
	testSizes("50%,", 101, 0)
	testSizes("10,50%,10", 101, 0)
	testSizes("10,*,10", 101, 0)
	testSizes("10,*,*,10", 101, 0)
	testSizes("*,*,*", 100, 0)
	testSizes("*,*,*", 20, 0)
	testSizes("*,*,*", 20, 1)
	testSizes("*,*,*,*", 20, 0)
	testSizes("*,*,*,*,*,*", 20, 0)
	testSizes("*,*,*,*,*,*", 21, 0)
	testSizes("*,*,*,*,*,*", 22, 0)
	testSizes("*,*,*,*,*,*", 24, 0)
	testSizes("*,*,*,*,*,*", 25, 0)
	testSizes("*,*,*,*,*,*", 26, 0)
	testSizes("*,*,*,*,*,*", 19, 1)
	testSizes("*,*,*,*,*,*", 20, 1)
	testSizes("*,*,*,*,*,*", 21, 1)
	testSizes("*,*,*,*,*,*", 22, 1)
	testSizes("*,*,*,*,*,*", 24, 1)
	testSizes("*,*,*,*,*,*", 25, 1)
	testSizes("*,*,*,*,*,*", 26, 1)
	testSizes("", 4, 0)
	testSizes("*,1", 4, 0)
	testSizes("*,2", 4, 0)
	testSizes("*,3", 4, 0)
	testSizes("*,4", 4, 0)
	testSizes("*,5", 4, 0)
	testSizes("*,10", 9, 0)
	testSizes("*,6", 4, 0)
	testSizes("*,7", 4, 0)


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
	lines.SpanSeparator = "|"
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
			if len(s) != i {
				err = true
				if ! printed {
					printed = true
					fmt.Println("ERR: \""+ str +"\"") }
				fmt.Println(">"+ s + "<") 
				fmt.Printf("^%"+ fmt.Sprint(i) +"v^\n"+
						"\tshould be: %v got: %v\n"+
						"\tsizes: %v\n", 
					"", i, len(s), 
					lines.parseSizes(lines.SpanMode, i, len(string(lines.SpanSeparator)))) } } 
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
			c = 2 }
		if len(s) - c != w {
			fmt.Printf("\tparseSizes: %#v\n",
				lines.parseSizes(lines.SpanMode, w-c, 1))
			fmt.Printf("\tmakeSections: %#v\n",
				lines.makeSections(str, w-c, 1))
			fmt.Println("    -> ERR") 
		} else {
			fmt.Println("    -> OK") } }
	lines.Border = "│┌─┐│└─┘"
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 20)
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 21)
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 22)
	testBorderedSize("o%SPANv%SPANe%SPANr%SPANf%SPANl%SPANo%SPANw", 23)

	// XXX BUG: these do not show overflow...
	//		...this is likely because the last sections is 0 length...
	lines.SpanMode = ""
	testBorderedSize("moo%SPANfoo", 7)
	lines.Border = ""
	testBorderedSize("moo%SPANfoo", 5)
}


