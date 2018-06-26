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
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const endl = "\r\n"

// JackCompiler is the actual compiler
type JackCompiler struct {
	inFile  *os.File
	outFile *os.File
	scanner *bufio.Scanner

	symbols symbolTable

	labelCounter int
}

func (this *JackCompiler) nextLabel() string {
	this.labelCounter++
	return strconv.Itoa(this.labelCounter)
}

func (this *JackCompiler) Init(inPath, outPath, xmlName string) {
	var err error
	this.inFile, err = os.OpenFile(inPath+xmlName, os.O_APPEND|os.O_RDWR, 0644)
	check(err)

	vmName := xmlName
	vmName = vmName[0:len(vmName)-len(filepath.Ext(vmName))] + ".vm"

	this.outFile, err = os.Create(outPath + vmName)
	check(err)

	this.scanner = bufio.NewScanner(this.inFile)
}

func (this *JackCompiler) Free() {
	this.inFile.Close()
	this.outFile.Close()
}
func (this *JackCompiler) Compile() {
	this.compileClass()
}

func (this *JackCompiler) nextToken() (bool, string, string) {
	var isTerminal bool
	if this.scanner.Scan() {
		t := strings.TrimSpace(this.scanner.Text())
		splt := strings.Split(t, "<")
		splt = strings.Split(splt[1], ">")
		if len(splt) == 2 {
			isTerminal = true
		} else {
			isTerminal = false
		}
		return isTerminal, splt[0], splt[1]
	}
	return false, "", ""
}

func (this *JackCompiler) writeLine(data string) {
	this.outFile.WriteString(data + endl)
}

func (this *JackCompiler) compileClass() {
	this.symbols = symbolTable{} //init
	this.nextToken()             //open class
	this.nextToken()             //class keyword

	_, _, className := this.nextToken()
	this.symbols.newClass(className)

	this.nextToken() //{

	this.compileClassVarDecs()
	this.compileSubroutineDecs()

	this.nextToken() //}
	this.nextToken() //close class
}

func (this *JackCompiler) compileClassVarDecs() {
	_, key, _ := this.nextToken()
	for key == "classVarDec" {
		this.compileClassVarDec()
		_, key, _ = this.nextToken()
	}
}

func (this *JackCompiler) compileClassVarDec() {
	_, _, keyword := this.nextToken() //field or static
	_, _, Type := this.nextToken()
	_, _, name := this.nextToken()
	switch keyword {
	case "field":
		this.symbols.addField(name, Type)
	case "static":
		this.symbols.addStatic(name, Type)
	}

	_, _, semicomma := this.nextToken()
	for semicomma == "," { //otherwise its ;
		_, _, name = this.nextToken()
		this.symbols.addField(name, Type)
		_, _, semicomma = this.nextToken()
	}
	this.nextToken() // close ClassVarDec
}

func (this *JackCompiler) compileSubroutineDecs() {
	this.compileSubroutineDec()
	_, key, _ := this.nextToken()
	for key == "subroutineDec" {
		this.compileSubroutineDec()
		_, key, _ = this.nextToken()
	}

	// no need to close SubroutineDecs
}

func (this *JackCompiler) compileSubroutineDec() {
	this.symbols.newFunction()
	_, _, funcType := this.nextToken()
	if funcType == "method" { //making sure we all know that "this" is here
		this.symbols.argsCtr++
	}
	this.nextToken() //return type
	_, _, funcName := this.nextToken()
	this.nextToken() //(
	this.nextToken() //open parameterList
	this.compileParameterList()
	this.nextToken() //)

	///////////////////////////
	//compileSubroutineBody():/
	///////////////////////////

	this.nextToken()              //open subroutineBody
	this.nextToken()              //{
	_, key, _ := this.nextToken() //varDec or statements
	for key == "varDec" {         //varDec or statements
		this.nextToken() //open var
		this.compileVarDec()
		_, key, _ = this.nextToken() //varDec or statements
	}

	this.writeLine("function " + this.symbols.currentClassName + "." + funcName +
		" " + strconv.Itoa(this.symbols.varsCtr))

	switch funcType {
	case "method":
		this.writeLine("push argument 0")
		this.writeLine("pop pointer 0") //set the scope to this class
	case "constructor":
		this.writeLine("push constant " + strconv.Itoa(this.symbols.fieldsCtr))
		this.writeLine("call Memory.alloc 1") //allocate memory to contain this class
		this.writeLine("pop pointer 0")       //set the scope to this class
	case "function": //nothing special
	}
	//continue compiling subroutineBody

	this.compileStatements()

	this.nextToken() // }
	this.nextToken() // close subRoutineBody (not decs!!)
	this.nextToken() // close subRoutineDec (not decs!!)
}

func (this *JackCompiler) compileParameterList() {
	var key, name, Type string
	_, key, Type = this.nextToken()
	for key != "/parameterList" {
		//_, _, Type := this.nextToken()
		_, _, name = this.nextToken()
		this.symbols.addArg(name, Type)
		_, key, Type = this.nextToken()
		if Type == "," {
			_, key, Type = this.nextToken()
		}
	}

	//close parameterList already read
}

func (this *JackCompiler) compileVarDec() {
	_, _, Type := this.nextToken()
	_, _, name := this.nextToken()

	this.symbols.addVar(name, Type)

	_, _, semicomma := this.nextToken()
	for semicomma == "," { //otherwise its ;
		_, _, name = this.nextToken()
		this.symbols.addVar(name, Type)
		_, _, semicomma = this.nextToken()
	}

	this.nextToken() //close varDec
}

func (this *JackCompiler) compileStatements() {
	_, key, _ := this.nextToken() //a statement or ends statements
	for key != "/statements" {
		switch key {
		case "letStatement":
			this.compileLetStatement()
		case "ifStatement":
			this.compileIfStatement()
		case "whileStatement":
			this.compileWhileStatement()
		case "doStatement":
			this.compileDoStatement()
		case "returnStatement":
			this.compileReturnStatement()
		}
		_, key, _ = this.nextToken() //a statement or ends statements
	}

	//close statements already read
}

func (this *JackCompiler) compileLetStatement() {
	this.nextToken() //let
	_, key, varName := this.nextToken()
	_, _, symbol := this.nextToken()
	if symbol == "[" {
		this.nextToken() //open expression
		this.push(key, varName)
		this.compileExpression()
		this.nextToken() //]
		this.writeLine("add")
		this.nextToken() //=
		this.nextToken() //open expression
		this.compileExpression()
		this.writeLine("pop temp 4") //ha i can do whatever i want. their our know rules
		this.writeLine("pop pointer 1")
		this.writeLine("push temp 4")
		this.writeLine("pop that 0")
	} else { //symbol == "="
		this.nextToken() //open expression
		this.compileExpression()
		this.pop(varName)
	}

	this.nextToken() // ;
	this.nextToken() //close letStatement

}

func (this *JackCompiler) compileIfStatement() {
	labelCounter := this.nextLabel()
	this.nextToken() //if
	this.nextToken() //(
	this.nextToken() //open expression
	this.compileExpression()
	this.nextToken()      //)
	this.writeLine("not") //saves a label
	this.writeLine("if-goto IF_FALSE" + labelCounter)
	this.nextToken() //{
	this.nextToken() //open statements
	this.compileStatements()
	this.nextToken()                //}
	_, _, value := this.nextToken() //close ifStatement or else
	if value == "else" {            //otherwise its close ifStatement
		this.writeLine("goto IF_END" + labelCounter)
		this.writeLine("label IF_FALSE" + labelCounter)
		this.nextToken() //{
		this.nextToken() //open statements
		this.compileStatements()
		this.nextToken() //}
		this.nextToken() // end ifStatement
		this.writeLine("label IF_END" + labelCounter)
	} else {
		this.writeLine("label IF_FALSE" + labelCounter) //no need for if end
	}

	//close ifStatement already read
}

func (this *JackCompiler) compileWhileStatement() {
	labelCounter := this.nextLabel()
	this.nextToken() //while
	this.nextToken() //(
	this.writeLine("label WHILE_BEG" + labelCounter)
	this.nextToken() //open expression
	this.compileExpression()
	this.nextToken()      //)
	this.writeLine("not") //saves a label
	this.writeLine("if-goto WHILE_END" + labelCounter)
	this.nextToken() //{
	this.nextToken() //open statements
	this.compileStatements()
	this.writeLine("goto WHILE_BEG" + labelCounter)
	this.nextToken() //}
	this.writeLine("label WHILE_END" + labelCounter)

	this.nextToken() //close whileStatement
}

func (this *JackCompiler) compileDoStatement() {
	this.nextToken() // do
	_, key, value := this.nextToken()
	Type, check, _ := this.symbols.find(value)

	if check == "function" { // inFile(~) or class.inFile(~)
		this.compileSubroutineCall(value) // no need to open for subroutinecall
	} else {
		this.push(key, value)
		this.nextToken()                 // .
		this.compileSubroutineCall(Type) // no need to open for subroutinecall
	}
	this.writeLine("pop temp 4") //throw away returned value (void)

	this.nextToken() // ;
	this.nextToken() // close doStatement
}

func (this *JackCompiler) compileReturnStatement() {
	this.nextToken()                //return
	_, _, value := this.nextToken() //close returnStatement or ;
	if value == ";" {
		this.writeLine("push constant 4") //this can be anything
	} else {
		//expression is allready opened --------this.nextToken() //open expression
		this.compileExpression()
		this.nextToken() // ;
	}
	this.writeLine("return")

	this.nextToken() //close returnStatement
}

func (this *JackCompiler) compileExpression() {
	this.nextToken() //open term
	this.compileTerm()
	_, key, value := this.nextToken()

	for key != "/expression" { //op
		op := value
		this.nextToken() //open term
		this.compileTerm()
		switch op {
		case "+":
			this.writeLine("add")
		case "-":
			this.writeLine("sub")
		case "*":
			this.writeLine("call Math.multiply 2")
		case "/":
			this.writeLine("call Math.divide 2")
		case "&amp;": //&
			this.writeLine("and")
		case "|":
			this.writeLine("or")
		case "&lt;": //<
			this.writeLine("lt")
		case "&gt;": //>
			this.writeLine("gt")
		case "=":
			this.writeLine("eq")
		}
		_, key, value = this.nextToken()
	}

	//close expression already read
}

func (this *JackCompiler) compileTerm() {
	_, key, value := this.nextToken()
	switch key {
	case "integerConstant", "keyword", "stringConstant":
		this.push(key, value)
	case "identifier":
		Type, check, _ := this.symbols.find(value)
		if check == "function" { // inFile(~) or class.inFile(~)
			this.compileSubroutineCall(value) // no need to open for subroutinecall
		} else {
			this.push(key, value) //this also pushes the var for function call
			_, _, symbolOrEnd := this.nextToken()
			if symbolOrEnd == "[" {
				this.nextToken() //open expression
				this.compileExpression()
				this.nextToken() //]
				this.writeLine("add")
				this.writeLine("pop pointer 1")
				this.writeLine("push that 0") //dereference pointer[1] (that == pointer[1])
			} else if symbolOrEnd == "." { //var.inFile(~)
				this.compileSubroutineCall(Type) // no need to open for subroutinecall
			} else {
				return //because we ate "/term" just now
			}
		}
	case "symbol":
		switch value {
		case "(":
			this.nextToken() //open expression
			this.compileExpression()
			this.nextToken() //)
		case "-":
			this.nextToken() //open term
			this.compileTerm()
			this.writeLine("neg")
		case "~":
			this.nextToken() //open term
			this.compileTerm()
			this.writeLine("not")
		}
	}

	this.nextToken() //close term
}

func (this *JackCompiler) compileSubroutineCall(funcNameOrType string) {
	var funcType string
	var funcName string
	numOfExpressions := 0
	_, _, value := this.nextToken() // "." or "("
	switch value {
	case "(": //inFile(~) - this.method()
		funcName = funcNameOrType
		funcType = this.symbols.currentClassName
		this.writeLine("push pointer 0")
		numOfExpressions++
	case ".": //class.inFile(~)
		funcType = funcNameOrType
		_, _, funcName = this.nextToken()
		this.nextToken() // "("
	default: //var.inFile(~)
		funcType = funcNameOrType
		funcName = value
		this.nextToken() // "("
		//var is already pushed to stack in calling function
		numOfExpressions++
	}

	this.nextToken() // open expressionList
	numOfExpressions += this.compileExpressionList()
	this.nextToken() // ")"
	this.writeLine("call " + funcType + "." + funcName + " " + strconv.Itoa(numOfExpressions))

	//no close because SubroutineCall doesn't exist
}

func (this *JackCompiler) compileExpressionList() int {
	var ctr int
	var value string
	_, key, _ := this.nextToken() // expression or close expressionList
	for key != "/expressionList" {
		ctr++
		this.compileExpression()
		_, key, value = this.nextToken() // expression or close expressionList
		if value == "," {
			_, key, _ = this.nextToken() // expression or close expressionList
		}
	}

	//close expressionList already read
	return ctr
}

func (this *JackCompiler) pop(value string) {
	_, where, offset := this.symbols.find(value)
	this.writeLine("pop " + where + " " + offset)
}

func (this *JackCompiler) push(Type, value string) {
	switch Type {
	case "integerConstant":
		this.writeLine("push constant " + value)
	case "stringConstant":
		this.writeLine("push constant " + strconv.Itoa(len(value)))
		this.writeLine("call String.new 1")
		for _, char := range value {
			this.writeLine("push constant " + strconv.Itoa(int(char)))
			this.writeLine("call String.appendChar 2")
		}
	case "keyword":
		switch value {
		case "false", "null":
			this.writeLine("push constant 0") //0 is both false and null
		case "true":
			this.writeLine("push constant 0") //-1 is bitwise "not" of 0, push can't handle negative
			this.writeLine("not")
		case "this":
			this.writeLine("push pointer 0")
		}
	case "identifier":
		_, where, offset := this.symbols.find(value)
		this.writeLine("push " + where + " " + offset)
	}

}
