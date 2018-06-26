/*
 *	Authors:
 *		Neria Tzidkani	209038876
 *		Gilad Weiss		206121527
 *
 *	Written Language:
 *		GoLang (aka Go!)
 *
 */
package main

import (
	"fmt"
	"bufio"
	"os"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"../stack"
	"flag"
)

/////////////////////////////////////////
//	Globals & Consts
/////////////////////////////////////////
var rawStr = ""
var wellDoneStr = ""
var depth = 0

// RegEx
var inlineCommentRegEx = regexp.MustCompile(`("[^"\\]*(?:\\.[^"\\]*)*")|//.*`)
var blockCommentRegEx = regexp.MustCompile(`("[^"\\]*(?:\\.[^"\\]*)*")|/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`)
var spaceMakerRegEx = regexp.MustCompile(`("[^"\\]*(?:\\.[^"\\]*)*")|` + wordsForSpaceRegex())
var spaceKillersRegEx = regexp.MustCompile(`("[^"\\]*(?:\\.[^"\\]*)*")|(?:\s+)`)

const tab = "\t"
const space = " "
const endl = "\r\n"

//var keywords = []string{"class", "constructor", "function", "method", "field", "static", "var", "int", "char", "boolean", "void", "true", "false", "null", "this", "let", "do", "if", "else", "while", "return"}
var symbols = []string{"{", "}", "(", ")", "[", "]", ".", ",", ";", "+", "-", "*", "/", "&", "|", "<", ">", "=", "~"}

var stk = stack.New()

/////////////////////////////////////////
//	Useful inline functions
/////////////////////////////////////////

// panic (aka exception) if something wrong
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// returns the string w/o "//" comment and afterwards
func killInlineComment(str string) string {
	return inlineCommentRegEx.ReplaceAllString(str, "$1")
	//	return str[:strings.Index(str, "//")]
}

func makeSpace(str string) string {
	return spaceMakerRegEx.ReplaceAllString(str, " $0 ")
}

func killSpace(str string) string {
	return spaceKillersRegEx.ReplaceAllString(str, " $1") //either with one space or with with string constant
}

func wordsForSpaceRegex() string {
	str := ""
	for _, symbol := range symbols {
		str += regexp.QuoteMeta(symbol) + "|"
	}
	str = str[:len(str)-1]
	return str
}

// returns the string w/o /*txt*/ comment.
// don't worry, /**/ inside strings are excluded [ i.e. "str/*str*/str"  ]
// see also at: https://stackoverflow.com/a/36779403
// this guy is a genius
func killBlockComment(str string) string {
	return blockCommentRegEx.ReplaceAllString(str, "$1")
}

func terminal(markup, value string) string {
	return strings.Repeat(tab, depth) + xmlize(markup, false) + value + xmlize(markup, true)
}

func xmlize(str string, isEnd bool) string {
	ret := "<"
	if isEnd {
		ret += "/"
	}
	return ret + str + ">"
}

//open new indent in the xml
func openXML(f *os.File, markup string) {
	stk.Push(markup)
	f.WriteString(strings.Repeat(tab, depth) + xmlize(markup, false) + endl)
	depth++
}

//close indent in xml
func closeXML(f *os.File) {
	markup := stk.Pop().(string)
	depth--
	f.WriteString(strings.Repeat(tab, depth) + xmlize(markup, true) + endl)
}

var isPrevIdentifier = false
var isPrevInteger = false

/*
 *	the used parsing algorithm is the LL(0) algorithm
 */
func trToMarkup(f *os.File, str string) {
	isCurrentIdentifier := false
	isCurrentInteger := false
	switch str {
	case "class":
		openXML(f, str)
		f.WriteString(terminal("keyword", str) + endl)
	case "field":
		fallthrough
	case "static": //definitely a class var
		openXML(f, "classVarDec")
		f.WriteString(terminal("keyword", str) + endl)

	case "null", "this", "true", "false":
		switch stk.Peek().(string) {
		case "expressionList", "returnStatement":
			openXML(f, "expression")
			openXML(f, "term")
		}

		fallthrough
	case "int":
		fallthrough
	case "char":
		fallthrough
	case "boolean":
		fallthrough
	case "else": // can only appear in if
		fallthrough
	case "void":
		f.WriteString(terminal("keyword", str) + endl)

	case ",":
		if stk.Peek().(string) == "term"{
			closeXML(f)
		}
		if stk.Peek().(string) == "expression"{
			closeXML(f)
			f.WriteString(terminal("symbol", str) + endl)
			openXML(f, "expression")
			openXML(f, "term")
		} else {
			f.WriteString(terminal("symbol", str) + endl)
		}
	case ".":
		f.WriteString(terminal("symbol", str) + endl)
	case ";":
		if stk.Peek().(string) == "term" {
			closeXML(f)
			closeXML(f)
		}
		if stk.Peek().(string) == "expression" {
			closeXML(f)
		}
		f.WriteString(terminal("symbol", str) + endl)
		closeXML(f)

	case "constructor":
		fallthrough
	case "function":
		fallthrough
	case "method":
		openXML(f, "subroutineDec")
		f.WriteString(terminal("keyword", str) + endl)

	case "(":
		switch stk.Peek().(string) {
		case "expressionList", "returnExpression":
			openXML(f, "expression")
			openXML(f, "term")
		}

		switch stk.Peek().(string) {
		case "subroutineDec":
			f.WriteString(terminal("symbol", str) + endl)
			openXML(f, "parameterList")
		case "doStatement":
			isPrevIdentifier = true //then fallthrough. this makes the parser behave as though a function is being called
			fallthrough
		case "term":
			if isPrevIdentifier {
				f.WriteString(terminal("symbol", str) + endl)
				openXML(f, "expressionList")
				//openXML(f, "expression")
				//openXML(f, "term")
			} else { // if ( (
				f.WriteString(terminal("symbol", str) + endl)
				openXML(f, "expression")
				openXML(f, "term")
			}
		default: //the rest behave the same
			f.WriteString(terminal("symbol", str) + endl)
			//todo: del for some
			switch stk.Peek().(string) {
			case "ifStatement", "whileStatement":
				openXML(f, "expression")
				openXML(f, "term")
			}
		}
	case ")":
		switch stk.Peek().(string) {
		case "parameterList":
			closeXML(f)
			//f.WriteString(terminal("symbol", str) + endl)
		case "term":
			closeXML(f)
			closeXML(f)
		}

		if stk.Peek().(string) == "expression" {
			closeXML(f)
		}
		if stk.Peek().(string) == "expressionList" {
			closeXML(f)
		}
		f.WriteString(terminal("symbol", str) + endl)
		if stk.Peek().(string) == "term"{//term still open for some reason
			closeXML(f)
		}

		isCurrentIdentifier = isPrevIdentifier
		isCurrentInteger = isPrevInteger

	case "{":
		head := stk.Peek().(string)
		switch head {
		case "subroutineDec":
			openXML(f, "subroutineBody")
			f.WriteString(terminal("symbol", str) + endl)
		case "whileStatement", "ifStatement": // if & while
			f.WriteString(terminal("symbol", str) + endl)
			openXML(f, "statements")
		case "class":
			f.WriteString(terminal("symbol", str) + endl)
		}
	case "}":
		if stk.Peek().(string) == "ifStatement" {
			closeXML(f)
		}
		switch stk.Peek().(string) {
		case "class", "subroutineDec":
			f.WriteString(terminal("symbol", str) + endl)
			closeXML(f)
		case "statements":
			closeXML(f)
			f.WriteString(terminal("symbol", str) + endl)
			if stk.Peek().(string) != "ifStatement" {
				closeXML(f) //body/if/else/while
			}
			switch stk.Peek().(string) {
			case "subroutineDec":
				closeXML(f) //dec
			}
		default:
			closeXML(f)
			f.WriteString(terminal("symbol", str) + endl)
		}

	case "[":
		f.WriteString(terminal("symbol", str) + endl)
		openXML(f, "expression")
		openXML(f, "term")
	case "]":
		closeXML(f)
		closeXML(f)
		f.WriteString(terminal("symbol", str) + endl)
	case "var":
		openXML(f, "varDec")
		f.WriteString(terminal("keyword", str) + endl)

	case "let", "if", "while", "do", "return":
		switch stk.Peek().(string) {
		case "ifStatement":
			closeXML(f)
		case "subroutineBody":
			openXML(f, "statements")
		}
		openXML(f, str+"Statement")
		f.WriteString(terminal("keyword", str) + endl)


	case "-", "~":
		switch stk.Peek().(string) {
		case "expressionList", "returnStatement":
			openXML(f, "expression")
			openXML(f, "term")
		}
		fallthrough

	case "<", ">", "&":
		if str == ">" {
			str = "&gt;"
		} else if str == "<" {
			str = "&lt;"
		} else if str == "&" {
			str = "&amp;"
		}
		fallthrough

	case "+", "*", "/", "|", "=":
		if stk.Peek().(string) == "term" {
			if str != "~" { //definitely unary if ~
				if str != "-" || isPrevIdentifier || isPrevInteger { //if '-' but prev was id or int then binary
					closeXML(f)
				}
			}
		}
		f.WriteString(terminal("symbol", str) + endl)

		if stk.Peek().(string) == "letStatement" {
			openXML(f, "expression")
		}
		openXML(f, "term")

	default:
		switch stk.Peek().(string) {
		case "expressionList", "returnStatement":
			openXML(f, "expression")
			openXML(f, "term")
		}
		if str[0] >= '0' && str[0] <= '9' {
			f.WriteString(terminal("integerConstant", str) + endl)
			isCurrentInteger = true
		} else if str[0] == '"' {
			f.WriteString(terminal("stringConstant", str[1:len(str)-1]) + endl)
		} else {
			f.WriteString(terminal("identifier", str) + endl)
			isCurrentIdentifier = true
		}
	}

	isPrevIdentifier = isCurrentIdentifier
	isPrevInteger = isCurrentInteger
}

func XMLgenerator(inpath, outpath string) {
	dir, err := ioutil.ReadDir(inpath)
	check(err)
	for _, f := range dir {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".jack" {
			infile, err := os.OpenFile(inpath+f.Name(), os.O_APPEND|os.O_RDWR, 0644)
			check(err)

			filename := f.Name()
			filename = filename[0:len(filename)-len(filepath.Ext(filename))]

			scanner := bufio.NewScanner(infile)

			for scanner.Scan() {
				rawStr += killInlineComment(scanner.Text()) + " "
			}
			infile.Close()

			/*
			This is how its done:
			1)	kill block comments
			2)	make spaces between saved words. i.e. {}()var etc'
			3)	kill too many spaces. one is enough
			*/
			wellDoneStr = killSpace(makeSpace(killBlockComment(rawStr)))
			rawStr = "" // something like de-allocation

			outfile, err := os.Create(outpath + filename + ".xml")
			check(err)

			countingStr := ""
			isCounting := false
			DBG := wellDoneStr
			DBG = DBG
			for _, word := range strings.Split(wellDoneStr, " ") {
				if word != "" {
					if isCounting {
						countingStr += " " + word
						if word[len(word)-1] == '"' {
							isCounting = false
							trToMarkup(outfile, countingStr)
							countingStr = ""
						}
					} else if word[0] == '"' && len(word) == 1 {
						isCounting = true
						countingStr += word
					} else if word[0] == '"' && !isCounting {
						if word[len(word)-1] == '"' {
							trToMarkup(outfile, word)
						} else {
							isCounting = true
							countingStr += word
						}
					} else {
						trToMarkup(outfile, word)
					}
				}
			}
			//closeXML(outfile)
			outfile.Close()
		}
	}
}

func parameretrsFromCLI() (string, string) {
	args := os.Args[1:]
	args = append(args, "", "")
	i := args[0]
	o := args[1]
	if i == "" || i[0] == '-' {
		i = ""
	}
	if o == "" || o[0] == '-' {
		o = ""
	}

	var input, output string

	flag.StringVar(&input, "i", "", "input folder path")
	flag.StringVar(&output, "o", "", "output folder path")
	flag.Parse()

	if i == "" && o == "" {
		i = input
		o = output
	}
	return i, o
}

func main() {
	inpath, outpath := parameretrsFromCLI()
	if inpath == "" {
		fmt.Println("enter path of jack files:")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		inpath = scanner.Text() + `\`
	}
	if outpath == "" {
		outpath = inpath
	}
	XMLgenerator(inpath, outpath)
	//main()
}

/*
func openXML(f *os.File, markup string, depth int) {
	f.WriteString(strings.Repeat(tab, depth) + xmlize(markup, false) + endl)
}

func closeXML(f *os.File, markup string, depth int) {
	f.WriteString(strings.Repeat(tab, depth) + xmlize(markup, true) + endl)
}

func isTerminal(str string) bool {
	for _, w := range keywords {
		if w == str {
			return true
		}
	}
	for _, w := range symbols {
		if w == str {
			return true
		}
	}
	if str[0] >= '0' && str[0] <= '9' {
		return true
	}
	return str[0] == '"'
}*/

/*
func translate(f *os.File, markup string) {
	depth := 0
	for _, w := range strings.Split(wellDoneStr, " ") {
		if !isTerminal(w) {
			openXML(f, w, depth)
			defer closeXML(f, w, depth)
		} else {

		}

	}

}
*/
