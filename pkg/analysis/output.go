package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var outputFormats = map[string]func(*Analysis) ([]byte, error){
	"json": (*Analysis).jsonOutput,
	"text": (*Analysis).textOutput,
}

func getOutputFormats() []string {
	formats := make([]string, 0, len(outputFormats))
	for format := range outputFormats {
		formats = append(formats, format)
	}
	return formats
}

func (a *Analysis) PrintOutput(format string) ([]byte, error) {
	outputFunc, ok := outputFormats[format]
	if !ok {
		return nil, fmt.Errorf("unsupported output format: %s. Available format %s", format, strings.Join(getOutputFormats(), ","))
	}
	return outputFunc(a)
}

func (a *Analysis) jsonOutput() ([]byte, error) {
	var problems int
	var status AnalysisStatus
	for _, result := range a.Results {
		problems += len(result.Error)
	}
	if problems > 0 {
		status = StateProblemDetected
	} else {
		status = StateOK
	}

	result := JsonOutput{
		Provider: a.AnalysisAIProvider,
		Problems: problems,
		Results:  a.Results,
		Errors:   a.Errors,
		Status:   status,
	}
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %v", err)
	}
	return output, nil
}

func saveError(f *os.File, text string) {
	if _, err := f.WriteString(text + "\n"); err != nil {
		fmt.Println("Failed to write ", err)
	}
}

func (a *Analysis) textOutput() ([]byte, error) {
	var output strings.Builder

	// Print the AI provider used for this analysis
	// output.WriteString(fmt.Sprintf("AI Provider: %s\n", color.YellowString(a.AnalysisAIProvider)))

	f, err := os.OpenFile("/tmp/k8sgpt-errors.list", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("Error OpenFile", err)
	}
	defer f.Close()

	if len(a.Errors) != 0 {
		output.WriteString("\n")
		output.WriteString(color.YellowString("Warnings : \n"))
		for _, aerror := range a.Errors {
			output.WriteString(fmt.Sprintf("- %s\n", color.YellowString(aerror)))
		}
	}
	if !a.ShortText {
		output.WriteString("\n")
	}
	if len(a.Results) == 0 {
		if !a.ShortText {
			output.WriteString(color.GreenString("No problems detected\n"))
		}
		return []byte(output.String()), nil
	}
	for n, result := range a.Results {
		if !a.ShortText {
			output.WriteString(fmt.Sprintf("\n------------------------------------------------------------------------------------\n%s Resource: %s, Parent: %s\n",
				color.CyanString("%d", n),
				color.YellowString(result.Name), color.CyanString(result.ParentObject)))
		}

		for _, err := range result.Error {
			saveError(f, err.Text)
			if a.ShortText {
				output.WriteString(fmt.Sprintf("%s, %s %s : %s\n", result.Namespace, result.Kind, result.ResourceName, err.Text))
			} else {
				output.WriteString(fmt.Sprintf("\n%s %s\n", color.RedString("Error:"), color.RedString(err.Text)))
				if err.KubernetesDoc != "" {
					output.WriteString(fmt.Sprintf("  %s %s\n", color.RedString("Kubernetes Doc:"), color.RedString(err.KubernetesDoc)))
				}
			}
		}
		if len(result.Details) > 7 {
			output.WriteString("\n" + color.GreenString(result.Details[7:]+"\n"))
		} else if len(result.Details) > 0 {
			output.WriteString("\n" + color.GreenString(result.Details+"\n"))
		}

	}
	return []byte(output.String()), nil
}
