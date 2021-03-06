package main

import (
	"braid/ast"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
	name    = kingpin.Arg("name", "Braid file to process.").Required().String()
)

// takes raw Braid code and returns valid Go code.
func Compile(input string, verbose bool) (string, error) {
	lines := strings.Split(input, "\n")

	// first we make a reader from the input, which is a string
	r := strings.NewReader(input)
	// then we parse the input into ast
	result, err := ast.ParseReader("", r)

	if err != nil {

		fmt.Println("ERROR:")

		list := err.(ast.ErrorLister).Errors()
		for _, err := range list {
			// for each error, get the internal error
			pe := err.(ast.ParserError)
			printError(pe, lines)
		}
		return "", err
	}

	if verbose {
		// print the input
		for i, el := range lines {
			fmt.Printf("%03d|%s\n", i+1, el)
		}
	}

	// print the ast
	a := result.(ast.Module)

	//fmt.Println("=", a.Print(0))

	env := ast.State{Env: make(map[string]ast.Type), UsedVariables: make(map[string]bool),
		Module: &a,
	}

	// infer types for the ast
	typedAst, err := a.Infer(&env, nil)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	//fmt.Println(a.ConcreteTypes)
	env.Module = &a

	if verbose {
		fmt.Println("=", typedAst.Print(0))

	}

	output, _ := json.MarshalIndent(env, "", "  ")
	if verbose {
		fmt.Println(string(output))
	}

	// print the compiled Go
	compiled, env := typedAst.Compile(env)
	//fmt.Println(result)
	return compiled, nil

}

func printError(pe ast.ParserError, lines []string) {

	// determine how many past lines to render
	start := pe.Pos()[0] - 1
	if pe.Pos()[0] >= 5 {
		start = pe.Pos()[0] - 5
	}

	// print those past lines up until the line of the error
	for i, el := range lines[start:pe.Pos()[0]] {
		offset := start
		fmt.Fprintf(os.Stderr, "%03d|%s\n", i+1+offset, el)
		//i += 1
	}

	// print the caret pointing to the position
	line := lines[pe.Pos()[0]-1]
	fmt.Fprintf(os.Stderr, "    ")
	pos := pe.Pos()[1]
	if pos > 0 {
		pos--
	}
	for _, el := range line[:pos] {
		if el == '\t' {
			fmt.Fprintf(os.Stderr, "----")
		} else {
			fmt.Fprintf(os.Stderr, "-")
		}
	}
	fmt.Fprintf(os.Stderr, "^\n\n")

	// print the actual error
	fmt.Fprintf(os.Stderr, "Line %d, character %d: ", pe.Pos()[0], pe.Pos()[1])
	fmt.Fprintln(os.Stderr, pe.InnerError())
}

func main() {
	kingpin.Parse()

	// if len(args) == 0 {
	// 	fmt.Printf("Must supply an argument of a file to compile, eg. `$ braid example.bd`\n")
	// 	return
	// }

	result, err := ioutil.ReadFile(*name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %s\n", *name, err.Error())
		return
	}

	file := string(result)
	compiled, cErr := Compile(file, *verbose)
	if cErr != nil {
		return
	}
	fmt.Print(compiled)

}
