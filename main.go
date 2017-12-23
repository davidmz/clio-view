//go:generate statik -src=./htdocs
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/davidmz/clio-view/statik"
	"github.com/rakyll/statik/fs"
)

type indexData struct {
	Version  string
	Archives []*Archive
}

func main() {
	var (
		listenPort    int
		archDirectory string
		showHelp      bool
		noBrowser     bool
	)

	fmt.Println("clio-view v.", version)
	fmt.Println("")

	flag.IntVar(&listenPort, "p", 5335, "TCP port to listen")
	flag.StringVar(&archDirectory, "d", ".", "directory with clio archives")
	flag.BoolVar(&showHelp, "h", false, "show help message and exit")
	flag.BoolVar(&noBrowser, "no-browser", false, "do not open browser, just start server")
	flag.Parse()

	if showHelp {
		flag.Usage()
		awaitEnter()
		return
	}

	var archives []*Archive
	var userNames []string

	// Reading dir
	fmt.Println("Looking for archives...")
	files, err := ioutil.ReadDir(archDirectory)
	if err != nil {
		printErrorAndExit(fmt.Sprintf("Cannot open directory '%s': %v\n", archDirectory, err))
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".zip") {
			continue
		}
		fullPath := filepath.Join(archDirectory, file.Name())

		arch, err := NewArchive(fullPath)
		if err != nil {
			fmt.Printf("Cannot open zip archive '%s': %v\n", file.Name(), err)
			continue
		}
		if len(archives) > 0 {
			fmt.Print(", ")
		} else {
			fmt.Print("Found: ")
		}
		fmt.Print(arch.UserName)
		archives = append(archives, arch)
		userNames = append(userNames, arch.UserName)
	}

	if len(archives) == 0 {
		printErrorAndExit(fmt.Sprintf("Cannot find any archives\n", archDirectory, err))
	}
	fmt.Println("")
	fmt.Println("")

	jsonNames, _ := json.Marshal(userNames)

	textToAppend := []string{
		`<script>var userNames = ` + string(jsonNames) + `;</script>`,
		`<script src="/main.js"></script>`,
		`<link href="/main.css" rel="stylesheet" type="text/css">`,
	}

	appendMW := appender(strings.Join(textToAppend, "\n"))

	statikFS, err := fs.New()
	if err != nil {
		printErrorAndExit(fmt.Sprintf("Cannot open embedded FS: %v\n", err))
	}

	var indexTpl *template.Template
	{
		file, err := statikFS.Open("/index.html")
		if err != nil {
			printErrorAndExit(fmt.Sprintf("Cannot open index template file: %v\n", err))
		}
		b, err := ioutil.ReadAll(file)
		file.Close()

		indexTpl, err = template.New("index").Parse(string(b))
		if err != nil {
			printErrorAndExit(fmt.Sprintf("Cannot parse index template: %v\n", err))
		}
	}

	staticFS := http.FileServer(statikFS)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			if err := indexTpl.Execute(w, indexData{version, archives}); err != nil {
				http.Error(w, "Cannot render index page", http.StatusInternalServerError)
			}
		} else {
			pathParts := strings.Split(r.URL.Path, "/")
			username := pathParts[1]
			hndlr := staticFS
			for _, a := range archives {
				if a.UserName == username {
					hndlr = appendMW(a)
					break
				}
			}
			hndlr.ServeHTTP(w, r)
		}
	})

	fmt.Printf("Starting server at http://localhost:%d/\n", listenPort)

	listenErrChan := make(chan error)
	go func() {
		listenErrChan <- http.ListenAndServe(fmt.Sprintf(":%d", listenPort), nil)
	}()
	go func() {
		select {
		case <-time.After(500 * time.Millisecond):
			if !noBrowser {
				openBrowser(fmt.Sprintf("http://localhost:%d/\n", listenPort))
			}
		case err := <-listenErrChan:
			listenErrChan <- err
		}
	}()
	err = <-listenErrChan
	printErrorAndExit(fmt.Sprintf("Cannot start server: %v\n", err))
}

// openBrowser opens system browser with given url.
// From https://stackoverflow.com/a/39324149
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func printErrorAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	awaitEnter()
	os.Exit(1)
}

func awaitEnter() {
	fmt.Fprintln(os.Stderr, "Press 'Enter' to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
