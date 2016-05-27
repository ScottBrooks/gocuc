package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ScottBrooks/gocuc"
	gherkin "github.com/cucumber/gherkin3/go"
)

func GenerateAstJson(in io.Reader, out io.Writer, pretty bool) (err error) {
	feature, err := gherkin.ParseFeature(in)
	if err != nil {
		return
	}
	var bytes []byte
	if pretty {
		bytes, err = json.MarshalIndent(feature, "", "  ")
	} else {
		bytes, err = json.Marshal(feature)
	}
	if err != nil {
		return
	}
	out.Write(bytes)
	fmt.Fprint(out, "\n")
	return
}

func loadWire(r *gocuc.Runner) error {
	wire_root := "features/step_definitions/"
	files, err := ioutil.ReadDir(wire_root)
	if err != nil {
		return err
	}

	for _, f := range files {
		matched, _ := filepath.Match("*.wire", f.Name())
		if matched {
			return r.LoadWire(wire_root + f.Name())
		}
	}
	return fmt.Errorf("No wire file found")
}

var wireHost = flag.String("host", "127.0.0.1", "Host running cucumber")
var wirePort = flag.Int("port", 8666, "Port cucumber is running on")
var outputMode = flag.String("output", "dots,junit,template", "Output printer")
var launchPath = flag.String("path", "", "Process to launch to run the tests with")
var launchArgs = flag.String("args", "", "Arguments to the process we are launching")
var launchDir = flag.String("dir", "", "Working directory to use for the launched process")

type featureFile struct {
	r    io.Reader
	name string
}

func destroyProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		log.Printf("Killing launched process")
		p, err := os.FindProcess(cmd.Process.Pid)
		if p != nil {
			p.Kill()
			_, err = p.Wait()
			log.Printf("Could not find launced process: %+v", err)
		} else {
			err = cmd.Wait()
			log.Printf("Could not find launced process: %+v", err)
		}
	}
}

func main() {
	flag.Parse()
	var readers []featureFile

	var cmd *exec.Cmd
	var timeout *time.Timer

	if *launchPath != "" {
		args := strings.Split(*launchArgs, ",")
		cmd = exec.Command(*launchPath, args...)
		cmd.Dir = *launchDir
		log.Printf("Launching: %s", *launchPath)
		err := cmd.Start()
		if err != nil {
			log.Fatal("Launch Path specifed, but unable to launch process: %s %+v %+v", *launchPath, args, err)
		}
		time.Sleep(time.Second)
		timeout = time.AfterFunc(time.Hour, func() {
			destroyProcess(cmd)
			panic("THIS TEST RAN FOR AN HOUR, LETS STOP NOW")
		})
	}
	r := gocuc.Runner{}

	err := loadWire(&r)
	if err != nil {
		r.SetWire(*wireHost, *wirePort)
	}

	for _, l := range strings.Split(*outputMode, ",") {
		err = r.AddLogger(l)
		if err != nil {
			log.Fatalf("Unable to set output logger: %+v", err)
		}
	}
	r.Init()

	if len(os.Args) <= 1 {
		readers = append(readers, featureFile{os.Stdin, "STDIN"})
	} else {
		for i := range flag.Args() {
			path := flag.Arg(i)
			fmt.Printf("Running test: %s\n", path)
			matches, err := filepath.Glob(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
				return
			}
			for _, m := range matches {
				file, err := os.Open(m)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %s\n", err)
					os.Exit(1)
					return
				}
				defer file.Close()
				name := filepath.Base(m)
				readers = append(readers, featureFile{file, name})
			}
		}
	}

	startTime := time.Now().UnixNano() / 1e6
	for i := range readers {
		//err := GenerateAstJson(readers[i], os.Stdout, true)
		err := r.RunFeature(readers[i].r, readers[i].name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
	}
	endTime := time.Now().UnixNano() / 1e6
	if os.Getenv("GHERKIN_PERF") != "" {
		fmt.Fprintf(os.Stderr, "%d\n", endTime-startTime)
	}
	r.Shutdown()

	destroyProcess(cmd)
	if timeout != nil {
		timeout.Stop()
	}

}
