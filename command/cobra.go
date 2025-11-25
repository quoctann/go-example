package command

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

/*
go run main.go greet --name Alice
go run main.go calc --op mul --a 5 --b 3
go run main.go list --filter app
go run main.go user add --username john --email john@example.com
go run main.go user list
go run main.go --help
*/

var rootCmd = &cobra.Command{
	Use:   "myapp",
	Short: "A simple CLI app",
	Long:  "A simple CLI application demonstrating Cobra",
}

var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Greet someone",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		fmt.Printf("Hello, %s!\n", name)
	},
}

var calcCmd = &cobra.Command{
	Use:   "calc",
	Short: "Simple calculator",
	Run: func(cmd *cobra.Command, args []string) {
		op, _ := cmd.Flags().GetString("op")
		a, _ := cmd.Flags().GetFloat64("a")
		b, _ := cmd.Flags().GetFloat64("b")

		var result float64
		switch op {
		case "add":
			result = a + b
		case "sub":
			result = a - b
		case "mul":
			result = a * b
		case "div":
			if b == 0 {
				log.Fatal("Cannot divide by zero")
			}
			result = a / b
		default:
			log.Fatal("Unknown operation:", op)
		}
		fmt.Printf("%.2f %s %.2f = %.2f\n", a, op, b, result)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List items",
	Run: func(cmd *cobra.Command, args []string) {
		filter, _ := cmd.Flags().GetString("filter")
		items := []string{"apple", "banana", "cherry", "apricot"}

		for _, item := range items {
			if filter == "" || strings.Contains(item, filter) {
				fmt.Println("  -", item)
			}
		}
	},
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management",
}

var userAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new user",
	Run: func(cmd *cobra.Command, args []string) {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		fmt.Printf("Added user: %s (%s)\n", username, email)
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Users:")
		fmt.Println("  - john (john@example.com)")
		fmt.Println("  - jane (jane@example.com)")
	},
}

func init() {
	// Greet command flags
	greetCmd.Flags().StringP("name", "n", "World", "Name to greet")

	// Calc command flags
	calcCmd.Flags().String("op", "add", "Operation: add, sub, mul, div")
	calcCmd.Flags().Float64("a", 0, "First number")
	calcCmd.Flags().Float64("b", 0, "Second number")

	// List command flags
	listCmd.Flags().String("filter", "", "Filter items")

	// User add command flags
	userAddCmd.Flags().String("username", "", "Username")
	userAddCmd.Flags().String("email", "", "Email address")
	userAddCmd.MarkFlagRequired("username")
	userAddCmd.MarkFlagRequired("email")

	// Add commands to user group
	userCmd.AddCommand(userAddCmd, userListCmd)

	// Add commands to root
	rootCmd.AddCommand(greetCmd, calcCmd, listCmd, userCmd)
}

func RunCobraCLI(isSkip bool) {
	if isSkip {
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
