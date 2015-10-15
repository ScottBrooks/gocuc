package gocuc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/textproto"
	"os"
	"strconv"
	"strings"

	"github.com/cucumber/gherkin3/go"
)

type endpoint struct {
	Host  string
	Port  int
	Conn  *textproto.Conn
	Error error
}

type Runner struct {
	Endpoint    endpoint
	Loggers     loggerSlice
	QuitOnError bool
}
type logger interface {
	Init() error
	Shutdown()
	BeforeStep(step *gherkin.Step)
	Success(step *gherkin.Step)
	Failure(step *gherkin.Step, err error)
	BeginScenario(def *gherkin.ScenarioDefinition)
	EndScenario(def *gherkin.ScenarioDefinition)

	Example(header *gherkin.TableRow, row *gherkin.TableRow)
}

type loggerSlice []logger

func (loggers loggerSlice) BeforeStep(step *gherkin.Step) {
	for _, l := range loggers {
		l.BeforeStep(step)
	}
}

func (loggers loggerSlice) Success(step *gherkin.Step) {
	for _, l := range loggers {
		l.Success(step)
	}
}
func (loggers loggerSlice) Failure(step *gherkin.Step, err error) {
	for _, l := range loggers {
		l.Failure(step, err)
	}
}
func (loggers loggerSlice) BeginScenario(def *gherkin.ScenarioDefinition) {
	for _, l := range loggers {
		l.BeginScenario(def)
	}
}
func (loggers loggerSlice) EndScenario(def *gherkin.ScenarioDefinition) {
	for _, l := range loggers {
		l.EndScenario(def)
	}
}
func (loggers loggerSlice) Example(header *gherkin.TableRow, row *gherkin.TableRow) {
	for _, l := range loggers {
		l.Example(header, row)
	}
}
func (loggers loggerSlice) Init() {
	for _, l := range loggers {
		l.Init()
	}
}
func (loggers loggerSlice) Shutdown() {
	for _, l := range loggers {
		l.Shutdown()
	}
}

func (ep *endpoint) connect() error {
	conn, err := textproto.Dial("tcp", fmt.Sprintf("%s:%d", ep.Host, ep.Port))
	if err != nil {
		ep.Error = err
		return err
	}
	ep.Conn = conn
	log.Printf("Connected to endpoint")
	return nil
}

func (ep *endpoint) StepMatches(text string) []interface{} {
	if ep.Error != nil {
		return []interface{}{}
	}
	ep.Conn.PrintfLine(fmt.Sprintf("[\"step_matches\", {\"name_to_match\":\"%s\"}]", text))
	result, err := ep.Conn.ReadLine()
	if err != nil {
		ep.Error = err
		return []interface{}{}
	}

	var m []interface{}
	err = json.Unmarshal([]byte(result), &m)
	if err != nil {
		ep.Error = err
		return []interface{}{}
	}

	if len(m[1].([]interface{})) == 0 {
		ep.Error = fmt.Errorf("No steps mached: %s", text)
		return []interface{}{}

	} else if m[0].(string) == "success" {
		return m[1].([]interface{})
	} else {
		msg := m[1].(map[string]interface{})
		ep.Error = fmt.Errorf(msg["message"].(string))
		return []interface{}{}
	}
}

func (ep *endpoint) Invoke(arg interface{}, interfaceTable interface{}) {
	if ep.Error != nil {
		return
	}

	var table *gherkin.DataTable

	if interfaceTable != nil {
		table = interfaceTable.(*gherkin.DataTable)
	}

	id := arg.(map[string]interface{})["id"].(string)

	args := []string{}
	for _, arg := range arg.(map[string]interface{})["args"].([]interface{}) {
		args = append(args, "\""+arg.(map[string]interface{})["val"].(string)+"\"")
	}

	var msg string
	if table != nil {
		rows := []string{}
		for _, r := range table.Rows {
			cols := []string{}
			for _, c := range r.Cells {
				cols = append(cols, "\""+c.Value+"\"")
			}
			rows = append(rows, "["+strings.Join(cols, ",")+"]")
		}

		if len(args) == 0 {
			msg = fmt.Sprintf("[\"invoke\", {\"id\": \"%s\", \"args\": [[%s]]}]\r\n", id, strings.Join(rows, ","))
		} else {
			msg = fmt.Sprintf("[\"invoke\", {\"id\": \"%s\", \"args\": [%s, [%s]]}]\r\n", id, strings.Join(args, ","), strings.Join(rows, ","))
		}

	} else {
		msg = fmt.Sprintf("[\"invoke\", {\"id\": \"%s\", \"args\": [%s]}]\r\n", id, strings.Join(args, ","))

	}

	ep.Conn.W.WriteString(msg)
	ep.Conn.W.Flush()
	result, err := ep.Conn.ReadLine()
	if err != nil {
		ep.Error = err
		return
	}

	var m []interface{}
	err = json.Unmarshal([]byte(result), &m)
	if err != nil {
		ep.Error = err
	}
	if m[0].(string) != "success" {
		msg := m[1].(map[string]interface{})
		ep.Error = fmt.Errorf("\n-----------\nMessage: %s\n----------\nException: %s", msg["message"].(string), msg["exception"].(string))
	}

}

func (ep *endpoint) BeginScenario() {
	if ep.Error != nil {
		return
	}

	ep.Conn.PrintfLine("[\"begin_scenario\"]")
	result, err := ep.Conn.ReadLine()
	if err != nil {
		ep.Error = err
		return
	}
	var m []interface{}
	err = json.Unmarshal([]byte(result), &m)
	if err != nil {
		ep.Error = err
		return
	}
	if m[0].(string) != "success" {
		ep.Error = fmt.Errorf("Error begining scenario")
	}
}

func (ep *endpoint) EndScenario() {
	if ep.Error != nil {
		return
	}

	ep.Conn.PrintfLine("[\"end_scenario\"]")
	result, err := ep.Conn.ReadLine()
	if err != nil {
		ep.Error = err
		return
	}
	var m []interface{}
	err = json.Unmarshal([]byte(result), &m)
	if err != nil {
		ep.Error = err
		return
	}
	if m[0].(string) != "success" {
		ep.Error = fmt.Errorf("Error begining scenario")
	}
}

func (r Runner) runStep(step gherkin.Step) (bool, error) {
	r.Loggers.BeforeStep(&step)
	args := r.Endpoint.StepMatches(step.Text)
	err := r.Endpoint.Error
	if err == nil {
		r.Endpoint.Invoke(args[0], step.Argument)
	}
	err = r.Endpoint.Error

	return err == nil, err
}

func (r Runner) runScenarioOutline(outline *gherkin.ScenarioOutline) bool {
	allStepsOk := true

	for _, ex := range outline.Examples {
		for _, row := range ex.TableBody {
			r.Loggers.BeginScenario(&outline.ScenarioDefinition)
			r.Loggers.Example(ex.TableHeader, row)
			r.Endpoint.BeginScenario()

			for _, stepTemplate := range outline.Steps {
				step := gherkin.Step{Node: stepTemplate.Node, Keyword: stepTemplate.Keyword}

				// Filter our text
				filteredStepText := stepTemplate.Text
				for idx, key := range ex.TableHeader.Cells {
					filteredStepText = strings.Replace(filteredStepText, "<"+key.Value+">", row.Cells[idx].Value, -1)
				}
				step.Text = filteredStepText

				// Filter our arguments
				if stepTemplate.Argument != nil {
					template := stepTemplate.Argument.(*gherkin.DataTable)
					dt := gherkin.DataTable{}
					for _, r := range template.Rows {
						newRow := gherkin.TableRow{Node: template.Node}
						for _, c := range r.Cells {
							filteredArgumentText := c.Value
							for idx, key := range ex.TableHeader.Cells {
								filteredArgumentText = strings.Replace(filteredArgumentText, "<"+key.Value+">", row.Cells[idx].Value, -1)
							}
							newRow.Cells = append(newRow.Cells, &gherkin.TableCell{Node: c.Node, Value: filteredArgumentText})
						}
						dt.Rows = append(dt.Rows, &newRow)
					}
					step.Argument = &dt
				}
				success, err := r.runStep(step)
				if success {
					r.Loggers.Success(&step)
					/*					if step.Argument != nil {
										logDataTable(step.Argument.(*gherkin.DataTable))
									}*/
				} else {
					r.Loggers.Failure(&step, err)
					allStepsOk = false
					if r.QuitOnError {
						return allStepsOk
					}
					break
				}
			}
			r.Endpoint.EndScenario()
			r.Loggers.EndScenario(&outline.ScenarioDefinition)
		}
	}
	return allStepsOk
}
func (r Runner) runScenario(scenario *gherkin.Scenario) bool {
	steps := scenario.Steps
	allStepsOk := true
	r.Loggers.BeginScenario(&scenario.ScenarioDefinition)
	r.Endpoint.BeginScenario()

	for _, step := range steps {
		success, err := r.runStep(*step)
		if success {
			r.Loggers.Success(step)
			/*			if step.Argument != nil {
						logDataTable(step.Argument.(*gherkin.DataTable))
					}*/
		} else {
			r.Loggers.Failure(step, err)
			allStepsOk = false
			break
		}
	}
	r.Endpoint.EndScenario()
	r.Loggers.EndScenario(&scenario.ScenarioDefinition)
	return allStepsOk
}
func (r Runner) RunFeature(in io.Reader) error {
	feature, err := gherkin.ParseFeature(in)
	if err != nil {
		return err
	}

	for _, sd := range feature.ScenarioDefinitions {
		ok := true
		switch sd.(type) {
		case *gherkin.ScenarioOutline:
			outline := sd.(*gherkin.ScenarioOutline)

			ok = r.runScenarioOutline(outline)

		case *gherkin.Scenario:
			scenario := sd.(*gherkin.Scenario)
			ok = r.runScenario(scenario)
		}
		if !ok && r.QuitOnError {
			log.Printf("break")
			break
		}
	}

	return nil
}

func (r *Runner) SetWire(host string, port int) {
	r.Endpoint.Host = host
	r.Endpoint.Port = port
}

func (r *Runner) LoadWire(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	ep := endpoint{}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		switch scanner.Text() {
		case "host:":
			scanner.Scan()
			ep.Host = scanner.Text()
		case "port:":
			scanner.Scan()
			port, err := strconv.Atoi(scanner.Text())
			if err != nil {
				return err
			}
			ep.Port = port
		}
	}
	r.Endpoint = ep
	return r.Endpoint.connect()
}

func (r *Runner) AddLogger(name string) error {
	switch name {
	case "dots":
		r.Loggers = append(r.Loggers, &dotsLogger{})
		return nil
	case "junit":
		r.Loggers = append(r.Loggers, &junitLogger{})
		return nil
	}

	return fmt.Errorf("Unknown logger: %s", name)
}

func (r *Runner) Shutdown() {
	r.Loggers.Shutdown()
}
func (r *Runner) Init() {
	r.Loggers.Init()
}

func logDataTable(table *gherkin.DataTable) {
	if table != nil {
		for _, r := range table.Rows {
			cols := []string{}
			for _, c := range r.Cells {
				cols = append(cols, "\""+c.Value+"\"")
			}
			log.Printf("\t\t[" + strings.Join(cols, ",") + "]\n")
		}
	}

}
