package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var resolve []*ResolveStruct

func LoadResolveIndex() error {
	b, err := os.ReadFile(RESOLVE_DIR + "/index.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &resolve)
	if err != nil {
		return err
	}
	for _, r := range resolve {
		fmt.Println("loading", r.Pattern, r.File)
		r.Re = regexp.MustCompile(r.Pattern)
	}
	return nil
}

type ResolveStruct struct {
	Pattern string `json:"pattern"`
	File    string `json:"file"`
	Re      *regexp.Regexp
}

func (r *ResolveStruct) GetText(matches []string) string {
	b, err := os.ReadFile(RESOLVE_DIR + "/" + r.File)
	if err != nil {
		fmt.Print(err)
		return ""
	}
	rc := string(b)
	for ix, m := range matches {
		if ix == 0 {
			continue
		}
		// fmt.Println("REPLACE", ix, m)
		b2 := strings.Replace(rc, fmt.Sprintf("{{%d}}", ix), m, -1)
		rc = b2
	}
	return rc
}

func (a *Analysis) GetResolutionText(output string) error {
	if len(a.Results) == 0 {
		return nil
	}
	for index, analysis := range a.Results {

		parsedText := ""
		ref := ""
		for _, failure := range analysis.Error {
			// fmt.Printf("TEXT = %+v \n", failure.Text)
			for _, r := range resolve {
				match := r.Re.FindStringSubmatch(failure.Text)
				if len(match) > 1 {
					parsedText += r.GetText(match)
					ref = r.File
				}

			}
			break
		}
		// fmt.Println("DETAILS", index, parsedText)
		analysis.Ref = ref
		analysis.Details = parsedText
		a.Results[index] = analysis
	}
	return nil
}
