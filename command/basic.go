package command

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

// basic command, not use lib

/*
go run main.go greet --name=Alice
go run main.go calc --op=mul --a=5 --b=3
go run main.go list --filter=app
*/

func RunBasic(isSkip bool) {
	if isSkip {
		return
	}

	// subcommand: greet
	greetCmd := flag.NewFlagSet("greet", flag.ExitOnError)
	greetName := greetCmd.String("name", "World", "Name to greet")

	// subcommand: calc
	calcCmd := flag.NewFlagSet("calc", flag.ExitOnError)
	calcOp := calcCmd.String("op", "add", "Operation: add, sub, mul, div")
	calcA := calcCmd.Float64("a", 0, "First operand")
	calcB := calcCmd.Float64("b", 0, "Second operand")

	// subcommand: list
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listFilter := listCmd.String("filter", "", "Filter items")

	if len(os.Args) < 2 {
		println("expected 'greet', 'calc' or 'list' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "greet":
		greetCmd.Parse(os.Args[2:])
		fmt.Printf("Hello, %s!\n", *greetName)

	case "calc":
		calcCmd.Parse(os.Args[2:])
		result := performCalc(*calcOp, *calcA, *calcB)
		// %.2f formats a float with exactly 2 decimal places
		// e.g., 1.0 -> "1.00", 1.23456 -> "1.23", 1.2 -> "1.20"
		fmt.Printf("Result: %.2f\n", result)

	case "list":
		listCmd.Parse(os.Args[2:])
		listCmd.Parse(os.Args[2:])
		items := []string{"apple", "banana", "cherry"}
		for _, item := range items {
			if *listFilter == "" || strings.Contains(item, *listFilter) {
				fmt.Println("  -", item)
			}
		}

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Println("unknown command", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func performCalc(op string, a, b float64) float64 {
	switch op {
	case "add":
		return a + b
	case "sub":
		return a - b
	case "mul":
		return a * b
	case "div":
		if b == 0 {
			log.Fatal("Cannot divide by zero")
		}
		return a / b
	default:
		log.Fatal("Unknown operation")
		return 0
	}
}

func printUsage() {
	fmt.Println(`
Usage: myapp <command> [options]

Commands:
  greet    Greet someone
    --name  Name to greet (default: World)

  calc     Simple calculator
    --op    Operation: add, sub, mul, div (default: add)
    --a     First number
    --b     Second number

  list     List items
    --filter  Filter items by text

  help     Show this help message

Examples:
  myapp greet --name=Alice
  myapp calc --op=mul --a=5 --b=3
  myapp list --filter=app`)
}
