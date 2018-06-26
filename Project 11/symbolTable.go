// Authors:
//	Neria Tzidkani
//	Gilad Weiss
//
// Written Language:
//	GoLang (aka Go!)
//
package main

import "strconv"

type symbol struct {
	Type string
	Num  int
}

type symbolTable struct {
	currentClassName string

	staticsCtr int
	fieldsCtr  int
	argsCtr    int
	varsCtr    int

	statics map[string]symbol
	fields  map[string]symbol
	args    map[string]symbol
	vars    map[string]symbol
}

func (this *symbolTable) addStatic(Name, Type string) {
	this.statics[Name] = symbol{Type, this.staticsCtr}
	this.staticsCtr++
}
func (this *symbolTable) addField(Name, Type string) {
	this.fields[Name] = symbol{Type, this.fieldsCtr}
	this.fieldsCtr++
}
func (this *symbolTable) addArg(Name, Type string) {
	this.args[Name] = symbol{Type, this.argsCtr}
	this.argsCtr++
}
func (this *symbolTable) addVar(Name, Type string) {
	this.vars[Name] = symbol{Type, this.varsCtr}
	this.varsCtr++
}

func (this *symbolTable) newClass(className string) {
	this.currentClassName = className

	this.statics = map[string]symbol{}
	this.fields = map[string]symbol{}
	this.args = map[string]symbol{}
	this.vars = map[string]symbol{}

	this.staticsCtr = 0
	this.fieldsCtr = 0
	this.argsCtr = 0
	this.varsCtr = 0

}
func (this *symbolTable) newFunction() {
	this.args = map[string]symbol{}
	this.vars = map[string]symbol{}

	this.argsCtr = 0
	this.varsCtr = 0
}

func (this *symbolTable) find(value string) (string, string, string) {
	/*order to check:
	local
	arguments
	fields
	static*/

	var found symbol
	var location string

	location = "local"
	found = this.vars[value]
	if found.Type == "" {
		location = "argument"
		found = this.args[value]
		if found.Type == "" {
			location = "this"
			found = this.fields[value]
			if found.Type == "" {
				location = "static"
				found = this.statics[value]
				if found.Type == "" { // its a function
					return "function", "function", "function"
				}
			}
		}
	}

	return found.Type, location, strconv.Itoa(found.Num) //finally
}
