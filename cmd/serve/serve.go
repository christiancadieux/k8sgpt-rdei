/*
Copyright 2023 The K8sGPT Authors.
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

package serve

import (
	"github.com/spf13/cobra"
)

var (
	port        string
	metricsPort string
	backend     string
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Runs k8sgpt as a gRPC server",
	Long:  `Runs k8sgpt as a server to allow for easy integration with other applications.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	// add flag for backend
	// ServeCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to run the server on")
	ServeCmd.Flags().StringVarP(&metricsPort, "metrics-port", "", "8081", "Port to run the metrics-server on")
	ServeCmd.Flags().StringVarP(&backend, "backend", "b", "openai", "Backend AI provider")
}
