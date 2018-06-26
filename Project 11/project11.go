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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func vmGenerator(inPath, outPath string) {
	dir, err := ioutil.ReadDir(inPath)
	check(err)
	for _, f := range dir {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".xml" {
			name := f.Name()
			var compiler JackCompiler
			compiler.Init(inPath, outPath, name)
			compiler.Compile()
			compiler.Free()
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
	vmGenerator(inpath, outpath)
}
