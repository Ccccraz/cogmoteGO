/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

type program struct {
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	Serve()
}

func (p *program) Stop(s service.Service) error {
	return nil
}

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		register()
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}

func register() {
	svcConfig := &service.Config{
		Name:        "cogmoteGOService",
		DisplayName: "cogmoteGOService",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Logger.Info(err.Error())
	}

	err = s.Install()
	if err != nil {
		fmt.Printf("Failed to install: %s\n", err.Error())
	}
}
