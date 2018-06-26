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

// counter for labels (we dont like colusions)
var condCounter int

/////////////////////////////////////////
//	Usefull inline functions
/////////////////////////////////////////

// panic (aka exception) if somthing wrong
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Translate D = X very usfull macro
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

// Graeter than, Less than, Equal (aka gt, lt, eq)
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
	case "local":
		fallthrough
	case "argument":
		fallthrough
	case "this":
		fallthrough
	case "that":
		fallthrough
	case "temp":
		fallthrough
	case "pointer":
		ret += pushMemory(args)
	}
	return ret + pushD()
}

func genLabel(label, filename string) string {
	return filename + "." + label
}

func genGoto(label, filename string) string {
	return "@" + genLabel(label, filename) + endl +
		"0;JMP"
}

/////////////////////////////////////////
//	MAIN
/////////////////////////////////////////

func main() {
	var data, path string
	fmt.Println("enter path of vm files:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	path = scanner.Text() + `\`
	dir, err := ioutil.ReadDir(path)
	check(err)

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
					data += binaryOp("+")
				case "sub":
					data += binaryOp("-")
				case "neg":
					data += unaryOp("-")
				case "eq":
					fallthrough
				case "gt":
					fallthrough
				case "lt":
					data += condition(words[0])
				case "and":
					data += binaryOp(`&`)
				case "or":
					data += binaryOp(`|`)
				case "not":
					data += unaryOp("!")
				case "push":
					data += pushSomething(words, filename)
				case "pop":
					data += popSomething(words, filename)
				}
			}

			data = strings.Replace(data, "\t", "", -1)

			outfile, err := os.Create(path + filename + ".asm")
			check(err)

			outfile.WriteString(data)
			infile.Close()
			outfile.Close()

		}
	}
}
