// Authors:
//	Neria Tzidkani
//	Gilad Weiss
//
// Written Language:
//	GoLang (aka Go!)
//
package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

/////////////////////////////////////////
//	Globals & Consts
/////////////////////////////////////////

// cpp-style end line
const endl = "\n"

// D is lazyness for "D"
const D = "D"

// A is lazyness for "A"
const A = "A"
const bootsrtap = "@256" + endl +
	"D=A" + endl +
	"@SP" + endl +
	"M=D" + endl +
	"//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////" +
	"// END OF BOOTSTRAPPING" + endl

// counter for labels (we dont like collisions)
var condCounter int
var callCounter int

/////////////////////////////////////////
//	Useful inline functions
/////////////////////////////////////////

// panic (aka exception) if something wrong
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Translate D = X very useful macro
func deqX(x string) string {
	return "@" + x + `
			D=A
			`
}

/////////////////////////////////////////
//	Operations Translation
/////////////////////////////////////////

// Binary operation such as add/sub
func binaryOp(op string) string {
	return popReg("D") + //after pop, A is pointing to Y in stack
		`A=A-1
		D=M` + op + `D
		M=D
		`
}

// Unary operation such as logical NOT, AND, OR
func unaryOp(op string) string { //expensive
	return popReg("D") +
		`D=` + op + `D
		` + pushD()
}

// Greater than, Less than, Equal (aka gt, lt, eq)
// uses labels
func condition(cond string) string {
	condCounter++
	cnt := strconv.Itoa(condCounter)
	return popReg("D") +
		popReg("A") +
		`D=A-D
		@T` + cnt + `
		D;` + "J" + strings.ToUpper(cond) + `
		D=0
		` + pushD() +
		`@F` + cnt + `
		0;JMP
		(T` + cnt + `)
		D=-1
		` + pushD() +
		`(F` + cnt + `)
		`
}

/////////////////////////////////////////
//	Pop Translations
/////////////////////////////////////////

// pop register (A or D)
func popReg(reg string) string { //A dies
	return `@SP
		M=M-1
		A=M
		` + reg + `=M
		`
}

// push D register macro
func pushD() string {
	return `@SP
		A=M
		M=D
		@SP
		M=M+1
		`
}

func popSomething(args []string, filename string) string {
	ret := deqX(args[2]) + "@"
	switch args[1] {
	case "static":
		//yes, overwrite ret. its fine
		ret = "@" + filename + "." + args[2] + endl +
			"D=A" + endl
	case "local":
		ret += "LCL" + endl
	case "argument":
		ret += "ARG" + endl
	case "this":
		fallthrough
	case "that":
		ret += strings.ToUpper(args[1]) + endl
	case "temp":
		ret += "5" + endl
	case "pointer":
		if args[2] == "0" {
			ret += "THIS" + endl
		} else {
			ret += "THAT" + endl
		}
		ret += "D=A" + endl
	}

	if args[1] != "temp" && args[1] != "pointer" {
		ret += `A=M
				`
	}
	if args[1] != "pointer" {
		ret += "D=A+D" + endl
	}

	return ret +
		`@R13
		M=D
		` + popReg("D") + //popD overwrites A
		`@R13
		A=M
		M=D
		`
}

/////////////////////////////////////////
//	Push Translations
/////////////////////////////////////////

// Group no. 1 in teacher docs.
// Basically, push X Y where X is local/argument/this/that/temp/pointer and Y is integer
func pushMemory(args []string) string {
	ret := "@"
	switch args[1] {
	case "local":
		ret += "LCL" + endl
	case "argument":
		ret += "ARG" + endl
	case "this", "that":
		ret += strings.ToUpper(args[1]) + endl
	case "temp":
		ret += "5" + endl

	case "pointer":
		if args[2] == "0" {
			ret += "THIS" + endl
		} else {
			ret += "THAT" + endl
		}
	}
	//D=*(A+D)
	if args[1] != "temp" && args[1] != "pointer" {
		ret += `A=M
				`
	}
	if args[1] != "pointer" {
		ret += "A=A+D" + endl
	}
	ret += `D=M
			`
	return ret
}

func pushSomething(args []string, filename string) string {
	// D = X
	ret := deqX(args[2])

	switch args[1] {
	case "constant":
		break //do nothing
	case "static": //overwrite ret, yes. this is good. overwrite. no mistaky here k? no ketchup either. just sauce. raw sauce
		ret = "@" + filename + "." + args[2] + endl +
			"D=M" + endl
	case "local", "argument", "this", "that", "temp", "pointer":
		ret += pushMemory(args)
	}
	return ret + pushD()
}

func genLabel(label, filename string) string {
	return filename + "." + label
}

//1. add k to stack pointer
//2. from current stack location, put 0 until original stack location
//3. profit
func pushK0(k int) string {
	ret :=
		"@" + strconv.Itoa(k) + endl +
			`D=A
			@SP
			M=M+D
			A=M` + endl
	for i := 0; i < k; i++ {
		ret +=
			`A=A-1
			M=0
			`
	}
	return ret
}

//overwrites D and A
func pushScope(name string) string {
	return "@" + name + endl +
		"D=M" + endl +
		pushD()
}

func popScope(scope string, offset string) string {
	return `
			@` + offset + `
			D=A
			@LCL
			A=M-D
			D=M
			@` + scope + `
			M=D
`
}

func branchOps(words []string, filename string) string {
	var data string
	switch words[0] {
	case "label":
		data = "(" + genLabel(words[1], filename) + ")" + endl
	case "goto":
		data =
			"@" + genLabel(words[1], filename) + endl +
				"0;JMP" + endl
	case "if-goto":
		// jump occurs if the stack isn't ZERO
		data = popReg("D") + `@` + genLabel(words[1], filename) + `
						D;JNE
						`
	case "function":
		k, _ := strconv.Atoi(words[2])
		data =
			"(" + words[1] + ")" + endl +
				pushK0(k)

	case "call":
		callCounter++
		RA := "RA" + strconv.Itoa(callCounter)
		offset, _ := strconv.Atoi(words[2])
		offset += 5
		data =
			"@" + RA + endl +
				"D=A" + endl +
				pushD() +
				pushScope("LCL") +
				pushScope("ARG") +
				pushScope("THIS") +
				pushScope("THAT") +
				`@SP
				D=M
				@` + strconv.Itoa(offset) + endl +
				`D=D-A
				@ARG
				M=D
				@SP
				D=M
				@LCL
				M=D` + endl +
				"@" + words[1] + endl +
				"0;JMP" + endl +
				"(" + RA + ")" + endl
	case "return":
		data = `@LCL
				D=M
				@5
				A=D-A
				D=M
				@R15
				M=D
` +
			popSomething([]string{"pop ", "argument", "0"}, "") +
			`@ARG
			D=M+1
			@SP
			M=D` +
			popScope("THAT", "1") +
			popScope("THIS", "2") +
			popScope("ARG", "3") +
			popScope("LCL", "4") +
			`@R15
			A=M
			0;JMP
		`
	}
	return data
}

func main() {
	var data, path string
	fmt.Println("enter path of vm files:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	path = scanner.Text() + `\`
	dir, err := ioutil.ReadDir(path)
	check(err)

	outfile, err := os.Create(path + filepath.Base(path) + ".asm")
	check(err)
	defer outfile.Close()
	outfile.WriteString(bootsrtap)

	for _, f := range dir {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".vm" {
			infile, err := os.OpenFile(path+f.Name(), os.O_APPEND|os.O_RDWR, 0644)
			check(err)

			filename := f.Name()
			filename = filename[0 : len(filename)-len(filepath.Ext(filename))]

			scanner := bufio.NewScanner(infile)

			for scanner.Scan() {
				words := strings.Split(scanner.Text(), " ")
				if strings.HasPrefix(words[0], `//`) {
					continue
				}
				switch words[0] {
				case "add":
					data = binaryOp("+")
				case "sub":
					data = binaryOp("-")
				case "neg":
					data = unaryOp("-")
				case "eq", "gt", "lt":
					data = condition(words[0])
				case "and":
					data = binaryOp(`&`)
				case "or":
					data = binaryOp(`|`)
				case "not":
					data = unaryOp("!")
				case "push":
					data = pushSomething(words, filename)
				case "pop":
					data = popSomething(words, filename)
				case "function":
					if words[1] == "Sys.init" {
						outfile.Seek(17, io.SeekStart)                                                                          //17 is len of bootstrap without "call Sys.init 0"
						outfile.WriteString(strings.Replace(branchOps([]string{"call", "Sys.init", "0"}, "Sys"), "\t", "", -1)) //ich
						outfile.Seek(0, io.SeekEnd)
					}
					fallthrough
				case "label", "goto", "if-goto", "call", "return":
					data = branchOps(words, filename)
				default:
					continue //ignore
				}
				data = strings.Replace(data, "\t", "", -1)
				outfile.WriteString(data)
			}
			infile.Close()
		}
	}
}
