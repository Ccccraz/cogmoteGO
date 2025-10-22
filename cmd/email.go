/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Ccccraz/cogmoteGO/internal/keyring"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// emailCmd represents the email command
var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Store email credentials in the system keyring",
	Long:  "Prompt for email credentials and SMTP settings, then save the secrets to the keyring and persist the settings to the config file.",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter email address: ")
		emailAddress, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read email address: %v\n", err)
			return
		}
		emailAddress = strings.TrimSpace(emailAddress)
		if emailAddress == "" {
			fmt.Fprintln(os.Stderr, "email address cannot be empty")
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

		fmt.Print("Enter service address: ")
		serviceAddress, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read service address: %v\n", err)
			return
		}
		serviceAddress = strings.TrimSpace(serviceAddress)
		if serviceAddress == "" {
			fmt.Fprintln(os.Stderr, "service address cannot be empty")
			return
		}

		fmt.Print("Enter service port: ")
		portInput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read service port: %v\n", err)
			return
		}
		portInput = strings.TrimSpace(portInput)
		smtpPort, err := strconv.Atoi(portInput)
		if err != nil || smtpPort <= 0 || smtpPort > 65535 {
			fmt.Fprintln(os.Stderr, "service port must be a number between 1 and 65535")
			return
		}

		if err := keyring.SaveCredentials(emailAddress, password); err != nil {
			fmt.Fprintf(os.Stderr, "failed to store credentials: %v\n", err)
			return
		}

		viper.Set("email.send_email", emailAddress)
		viper.Set("email.smtp_host", serviceAddress)
		viper.Set("email.smtp_port", smtpPort)

		if err := viper.WriteConfig(); err != nil {
			configPath := viper.ConfigFileUsed()
			if configPath == "" {
				fmt.Fprintf(os.Stderr, "failed to save configuration: %v\n", err)
				return
			}
			if writeErr := viper.WriteConfigAs(configPath); writeErr != nil {
				fmt.Fprintf(os.Stderr, "failed to save configuration: %v\n", writeErr)
				return
			}
		}

		fmt.Println("email credentials saved to system keyring")
		fmt.Println("email configuration updated")
	},
}

func init() {
	rootCmd.AddCommand(emailCmd)
}
