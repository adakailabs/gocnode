/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/adakailabs/gocnode/runner"
	"github.com/spf13/cobra"
)

var id int
var isProducer bool

// startNodeCmd represents the start command
var startNodeCmd = &cobra.Command{
	Use:   "start-node",
	Short: "Start a cardano node",
	Long:  `Start a cardano node, relay or producer, based on the passed pool configuration and ID and is-producer flags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := runner.NewCardanoNodeRunner(conf, id, isProducer)
		if err != nil {
			return err
		}
		return r.StartCnode()
	},
}

// startNodeCmd represents the start command
var startPrometheus = &cobra.Command{
	Use:   "start-prometheus",
	Short: "Start prometheus for monitoring a cardano pool",
	Long:  `Start prometheus for monitoring a cardano pool, based on the passed pool configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := runner.NewPrometheusRunner(conf, id, isProducer)
		if err != nil {
			return err
		}
		return r.StartPrometheus()
	},
}

func init() {
	rootCmd.AddCommand(startNodeCmd)
	rootCmd.AddCommand(startPrometheus)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	startNodeCmd.PersistentFlags().IntVarP(&id, "id", "i", 0, "relay id")
	startNodeCmd.PersistentFlags().BoolVarP(&isProducer, "is-producer", "p", false, "starts this node as a producer")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startNodeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
