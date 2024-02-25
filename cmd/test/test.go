package test

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/k8sgpt-ai/k8sgpt/pkg/analysis"
	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/spf13/cobra"
	"os"
)

var (
	text      string
	output    string
	namespace string
	resource  string
)

var TestCmd = &cobra.Command{
	Use:     "test",
	Aliases: []string{"test"},
	Short:   "test",
	Long:    `test`,
	Run: func(cmd *cobra.Command, args []string) {

		if text == "" {
			color.Red("-t text -n namespace -r resource")
			os.Exit(1)
		}

		err := analysis.LoadResolveIndex()
		if err != nil {
			fmt.Println("loadResolve err", err)
			return
		}

		// AnalysisResult configuration
		ana, err := analysis.TestAnalysis()

		failures := []common.Failure{}
		failures = append(failures, common.Failure{Text: text})

		var currentAnalysis = common.Result{
			Namespace:    namespace,
			ResourceName: resource,
			Kind:         "Test",
			Name:         namespace + "/" + resource,
			Error:        failures,
		}

		ana.Results = append(ana.Results, currentAnalysis)

		err = ana.GetResolutionText(output, true)

		// print results
		output, err := ana.PrintOutput("text")
		if err != nil {
			color.Red("Error: %v", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
	},
}

func init() {

	// namespace flag
	TestCmd.Flags().StringVarP(&text, "text", "t", "", "text to analyze")
	TestCmd.Flags().StringVarP(&namespace, "namespace", "n", "namespace1", "Namespace")
	TestCmd.Flags().StringVarP(&resource, "resource", "r", "pod1", "Resource Name")
	// no cache flag

}
