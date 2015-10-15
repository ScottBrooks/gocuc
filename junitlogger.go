package gocuc

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cucumber/gherkin3/go"
)

type testProperty struct {
	Name  string
	Value string
}

type testCase struct {
	Name    string  `xml:"name,attr"`
	Skipped bool    `xml:"skipped,omitempty"`
	Failed  string  `xml:"failure,omitempty"`
	Time    float64 `xml:"time,attr"`
}

type testSuite struct {
	Failures   int            `xml:"failures,attr"`
	Errors     int            `xml:"errors,attr"`
	Skipped    int            `xml:"skipped,attr"`
	Tests      int            `xml:"tests,attr"`
	Name       string         `xml:"name,attr"`
	Properties []testProperty `xml:"properties"`
	TestCases  []testCase     `xml:"testcase"`
}

type testSuites struct {
	XMLName  xml.Name    `xml:"testsuites"`
	Failures int         `xml:"failures,attr"`
	Errors   int         `xml:"errors,attr"`
	Tests    int         `xml:"tests,attr"`
	Suite    []testSuite `xml:"testsuite"`
}

type junitLogger struct {
	w      io.WriteCloser
	Suites testSuites
	s      *testSuite
	ts     time.Time
}

func (t *testSuite) updateStats() {
	for _, tc := range t.TestCases {
		t.Tests++
		if tc.Skipped {
			t.Skipped += 1
		} else if tc.Failed != "" {
			t.Failures++
		}
	}

}

func (l *junitLogger) Init() error {
	f, err := os.Create("TEST-all.xml")
	if err != nil {
		return err
	}
	l.w = f
	return nil
}

func (l *junitLogger) updateStats() {
	for i := range l.Suites.Suite {
		l.Suites.Suite[i].updateStats()
		l.Suites.Failures += l.Suites.Suite[i].Failures
		l.Suites.Errors += l.Suites.Suite[i].Errors
		l.Suites.Tests += l.Suites.Suite[i].Tests
	}
}

func (l junitLogger) Shutdown() {
	l.updateStats()
	data, err := xml.MarshalIndent(l.Suites, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	l.w.Write(data)
	l.w.Close()
}

func (l *junitLogger) BeforeStep(step *gherkin.Step) {
	l.ts = time.Now()
}

func (l junitLogger) Success(step *gherkin.Step) {
	runTime := time.Now().Sub(l.ts)
	l.s.TestCases = append(l.s.TestCases, testCase{Name: step.Text, Time: runTime.Seconds()})
	fmt.Printf(".")
}
func (l *junitLogger) Failure(step *gherkin.Step, err error) {
	runTime := time.Now().Sub(l.ts)
	l.s.TestCases = append(l.s.TestCases, testCase{Name: step.Text, Failed: err.Error(), Time: runTime.Seconds()})
}

func (l *junitLogger) BeginScenario(def *gherkin.ScenarioDefinition) {
	l.s = &testSuite{Name: def.Name}
}
func (l *junitLogger) EndScenario(def *gherkin.ScenarioDefinition) {
	l.Suites.Suite = append(l.Suites.Suite, *l.s)
}

func (l junitLogger) Example(header *gherkin.TableRow, row *gherkin.TableRow) {
	fmt.Printf("Scenario Example: ")
	for idx, h := range header.Cells {
		fmt.Printf("%s = %s ", h.Value, row.Cells[idx].Value)
	}
	fmt.Printf("\n")
}
