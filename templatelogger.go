package gocuc

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"time"

	"github.com/cucumber/gherkin3/go"
)

type templateLogger struct {
	w      io.WriteCloser
	Suites testSuites
	s      *testSuite
	ts     time.Time
	tmpl   *template.Template
}

func (l *templateLogger) Init() error {
	masterTemplate := template.Must(template.ParseFiles("logger.tmpl"))
	l.tmpl = masterTemplate
	f, err := os.Create("output.html")
	if err != nil {
		return err
	}
	l.w = f
	return nil
}

func (l *templateLogger) updateStats() {
	for i := range l.Suites.Suite {
		l.Suites.Suite[i].updateStats()
		l.Suites.Failures += l.Suites.Suite[i].Failures
		l.Suites.Errors += l.Suites.Suite[i].Errors
		l.Suites.Tests += l.Suites.Suite[i].Tests
	}
}

func (l templateLogger) Shutdown() {
	l.updateStats()

	err := l.tmpl.Execute(l.w, l.Suites)
	if err != nil {
		log.Fatal(err)
	}
	l.w.Close()
}

func (l *templateLogger) BeginFeature(name string) {
	l.Suites.Name = name
}

func (l *templateLogger) BeforeStep(step *gherkin.Step) {
	l.ts = time.Now()
}

func (l templateLogger) Success(step *gherkin.Step) {
	runTime := time.Now().Sub(l.ts)
	l.s.TestCases = append(l.s.TestCases, testCase{Name: fmt.Sprintf("%04d: %s", len(l.s.TestCases)+1, step.Text), ClassName: l.Suites.Name, Time: runTime.Seconds()})
	fmt.Printf(".")
}
func (l *templateLogger) Failure(step *gherkin.Step, err error) {
	runTime := time.Now().Sub(l.ts)
	l.s.TestCases = append(l.s.TestCases, testCase{Name: fmt.Sprintf("%04d: %s", len(l.s.TestCases)+1, step.Text), ClassName: l.Suites.Name, Failed: err.Error(), Time: runTime.Seconds()})
}

func (l *templateLogger) BeginScenario(def *gherkin.ScenarioDefinition) {
	l.s = &testSuite{Name: def.Name}
}
func (l *templateLogger) EndScenario(def *gherkin.ScenarioDefinition) {
	l.Suites.Suite = append(l.Suites.Suite, *l.s)
}

func (l templateLogger) Example(header *gherkin.TableRow, row *gherkin.TableRow) {
	fmt.Printf("Scenario Example: ")
	for idx, h := range header.Cells {
		fmt.Printf("%s = %s ", h.Value, row.Cells[idx].Value)
	}
	fmt.Printf("\n")
}
