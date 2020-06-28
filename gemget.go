package main

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/schollz/progressbar/v3"
	flag "github.com/spf13/pflag"
)

//var cert = flag.String("cert", "", "Not implemented.")
//var key = flag.String("key", "", "Not implemented.")

var insecure = flag.BoolP("insecure", "i", false, "Skip checking the cert")
var dir = flag.StringP("directory", "d", ".", "The directory where downloads go")
var output = flag.StringP("output", "o", "", "Output path, for when there is only one URL.\n'-' means stdout and implies --quiet.\nIt overrides --directory.")
var errorSkip = flag.BoolP("skip", "s", false, "Move to the next URL when one fails.")
var exts = flag.BoolP("add-extension", "e", false, "Add .gmi extensions to gemini files that don't have it, like directories.")
var quiet bool // Set in main, so that it can be changed later if needed
var numRedirs = flag.UintP("redirects", "r", 5, "How many redirects to follow before erroring out.")

func fatal(format string, a ...interface{}) {
	urlError(format, a...)
	os.Exit(1)
}

func urlError(format string, a ...interface{}) {
	if strings.HasPrefix(format, "\n") {
		format = "*** " + format[:len(format)-1] + " ***\n"
	} else {
		format = "*** " + format + " ***\n"
	}
	fmt.Fprintf(os.Stderr, format, a...)
	if !*errorSkip {
		os.Exit(1)
	}
}

func saveFile(resp *gemini.Response, u *url.URL) {
	var name string
	var savePath string

	if *output == "" {
		name := path.Base(u.Path) // Filename from URL
		if name == "/" || name == "." {
			// Domain is being downloaded, so there's no path/file
			name = u.Hostname()
		}
		if *exts && !(strings.HasSuffix(name, ".gmi") && strings.HasSuffix(name, ".gemini")) && (resp.Meta == "" || strings.HasPrefix(resp.Meta, "text/gemini")) {
			// It's a gemini file, but it doesn't have that extension - and the user wants them added
			name += ".gmi"
		}
		savePath = filepath.Join(*dir, name)
	} else {
		// There is an output path
		savePath = *output
	}

	f, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fatal("Error saving file %s", name)
	}
	defer f.Close()

	var written int64
	if quiet {
		written, err = io.Copy(f, resp.Body)
	} else {
		bar := progressbar.DefaultBytes(-1, "downloading")
		written, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
		fmt.Println()
	}
	if err != nil {
		fatal("Error saving file %s, %d bytes saved.", name, written)
	}
}

func _fetch(n uint, u *url.URL, client *gemini.Client) {
	uStr := u.String()
	resp, err := client.Fetch(uStr)
	if err != nil {
		urlError(err.Error())
		return
	}
	defer resp.Body.Close()

	// Validate status
	if resp.Status >= 60 {
		urlError("%s needs a certificate, which is not implemented yet.", uStr)
		return
	} else if gemini.SimplifyStatus(resp.Status) == 30 {
		if *numRedirs == 0 {
			urlError("%s redirects.", uStr)
			return
		}
		// Redirect
		if n == *numRedirs {
			urlError("%s redirected too many times.", uStr)
			return
		}
		// Follow the recursion
		redirect, err := url.Parse(resp.Meta)
		if err != nil {
			urlError("Redirect url %s couldn't be parsed.", resp.Meta)
			return
		}
		if !quiet {
			fmt.Printf("*** Redirected to %s ***\n", resp.Meta)
		}
		_fetch(n+1, u.ResolveReference(redirect), client)
	} else if resp.Status == 10 {
		urlError("%s needs input, which is not implemented yet. You should make the request manually with a URL query.", uStr)
	} else if gemini.SimplifyStatus(resp.Status) == 20 {
		// Output to stdout, otherwise save it to a file
		if *output == "-" {
			io.Copy(os.Stdout, resp.Body)
			return
		}
		saveFile(resp, u)
		return
	} else {
		urlError("%s returned status %d, skipping.", u, resp.Status)
	}
}

func fetch(u *url.URL, client *gemini.Client) {
	_fetch(1, u, client)
}

func main() {
	flag.BoolVarP(&quiet, "quiet", "q", false, "No output except for errors.")
	flag.Parse()

	// Validate urls
	if len(flag.Args()) == 0 {
		flag.Usage()
		fmt.Println("\n*** No URLs provided. ***")
		os.Exit(1)
	}
	urls := make([]*url.URL, len(flag.Args()))
	for i, u := range flag.Args() {
		parsed, err := url.Parse(u)
		if err != nil {
			urlError("%s could not be parsed.", u)
			continue
		}
		if len(parsed.Scheme) != 0 && parsed.Scheme != "gemini" {
			urlError("%s is not a gemini URL.", u)
			continue
		}
		if !strings.HasPrefix(u, "gemini://") {
			// Have to reparse due to the way the lib works
			parsed, err = url.Parse("gemini://" + u)
			if err != nil {
				urlError("Adding gemini:// to %s failed.", u)
			}
		}
		if parsed.Port() == "" {
			// Add port, gemini library requires it
			parsed.Host = net.JoinHostPort(parsed.Hostname(), "1965")
		}
		if parsed.Path == "" {
			// Add slash to the end of domains to prevent redirects
			parsed.Path = "/"
		}
		urls[i] = parsed
	}

	// Validate flags
	if len(flag.Args()) > 1 && *output != "" && *output != "-" {
		fatal("The output flag cannot be specified when there are multiple URLs, unless it is '-'.")
	}
	if *output == "-" {
		quiet = true
	}
	// Fetch each URL
	client := &gemini.Client{Insecure: *insecure}
	for _, u := range urls {
		if !quiet {
			fmt.Printf("*** Started %s ***\n", u)
		}
		fetch(u, client)
	}
}
