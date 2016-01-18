package gocuc

import (
	"fmt"

	"github.com/cucumber/gherkin3/go"
)

type dotsLogger struct {
	failed  bool
	message string
}

func (l *dotsLogger) Init() error {
	return nil
}
func (l dotsLogger) Shutdown() {

}

func (l dotsLogger) BeginFeature(name string) {
	fmt.Printf("Starting feature: %s\n", name)
}

func (l dotsLogger) BeforeStep(step *gherkin.Step) {
}

func (l dotsLogger) Success(step *gherkin.Step) {
	fmt.Printf(".")
}
func (l *dotsLogger) Failure(step *gherkin.Step, err error) {
	fmt.Printf("F")
	l.failed = true
	l.message = err.Error()
}

func (l *dotsLogger) BeginScenario(def *gherkin.ScenarioDefinition) {
	fmt.Printf("Scenario: %s\n", def.Name)
	l.failed = false
}
func (l dotsLogger) EndScenario(def *gherkin.ScenarioDefinition) {
	fmt.Printf("\n")
	if l.failed {
		fmt.Printf("Scenario failed: %s\n", l.message)
	}
}

func (l dotsLogger) Example(header *gherkin.TableRow, row *gherkin.TableRow) {
	fmt.Printf("\nScenario Example: ")
	for idx, h := range header.Cells {
		fmt.Printf("%s = %s ", h.Value, row.Cells[idx].Value)
	}
	fmt.Printf("\n")
}
