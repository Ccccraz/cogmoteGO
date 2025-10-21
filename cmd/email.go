/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Ccccraz/cogmoteGO/internal/keyring"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// emailCmd represents the email command
var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Store email credentials in the system keyring",
	Long:  "Prompt for an email username and password, then save them to the system keyring.",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter email username: ")
		username, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read username: %v\n", err)
			return
		}
		username = strings.TrimSpace(username)
		if username == "" {
			fmt.Fprintln(os.Stderr, "username cannot be empty")
			return
		}

		fmt.Print("Enter email password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read password: %v\n", err)
			return
		}
		password := strings.TrimSpace(string(passwordBytes))
		if password == "" {
			fmt.Fprintln(os.Stderr, "password cannot be empty")
			return
		}

		if err := keyring.SaveCredentials(username, password); err != nil {
			fmt.Fprintf(os.Stderr, "failed to store credentials: %v\n", err)
			return
		}

		fmt.Println("email credentials saved to system keyring")
	},
}

func init() {
	rootCmd.AddCommand(emailCmd)
}
