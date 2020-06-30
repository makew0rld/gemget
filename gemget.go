package main

import (
	"fmt"
	"io"
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
var header = flag.Bool("header", false, "Print out (even with --quiet) the response header to stdout in the format:\nHeader: <status> <meta>")

func fatal(format string, a ...interface{}) {
	urlError(format, a...)
	os.Exit(1)
}

func urlError(format string, a ...interface{}) {
	if strings.HasPrefix(format, "\n") {
		format = "Error: " + format[:len(format)-1] + "\n"
	} else {
		format = "Error: " + format + "\n"
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
		fatal("Couldn't save file %s: %v", name, err)
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
		fatal("Issue saving file %s, %d bytes saved: %v", name, written, err)
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

	if *header {
		fmt.Printf("Header: %d %s\n", resp.Status, resp.Meta)
	}

	// Validate status
	if resp.Status >= 60 {
		urlError("%s needs a certificate, which is not implemented yet.", uStr)
		return
	} else if gemini.SimplifyStatus(resp.Status) == 30 {
		if *numRedirs == 0 {
			urlError("This URL redirects but redirects are disabled: %s", uStr)
			return
		}
		// Redirect
		if n == *numRedirs {
			urlError("URL redirected too many times: %s", uStr)
			return
		}
		// Follow the recursion
		redirect, err := url.Parse(resp.Meta)
		if err != nil {
			urlError("Redirect URL %s couldn't be parsed.", resp.Meta)
			return
		}
		if !quiet {
			fmt.Fprintf(os.Stderr, "Info: Redirected to %s\n", resp.Meta)
		}
		_fetch(n+1, u.ResolveReference(redirect), client)
	} else if gemini.SimplifyStatus(resp.Status) == 10 {
		urlError("This URL needs input, you should make the request again manually: %s", uStr)
	} else if gemini.SimplifyStatus(resp.Status) == 20 {
		// Output to stdout, otherwise save it to a file
		if *output == "-" {
			io.Copy(os.Stdout, resp.Body)
			return
		}
		saveFile(resp, u)
		return
	} else {
		// Any sort of invalid status code will likely be caught by go-gemini, but this is here just in case
		urlError("URL returned status %d, skipping: %s", resp.Status, u)
	}
}

func fetch(u *url.URL, client *gemini.Client) {
	_fetch(1, u, client)
}

func main() {
	flag.BoolVarP(&quiet, "quiet", "q", false, "No info strings will be printed. Note that normally infos are printed to stderr, not stdout.")
	flag.Parse()

	// Validate urls
	if len(flag.Args()) == 0 {
		flag.Usage()
		fmt.Println("\nNo URLs provided.")
		os.Exit(1)
	}
	urls := make([]*url.URL, len(flag.Args()))
	for i, u := range flag.Args() {

		parsed, err := url.Parse(u)
		if err != nil {
			urlError("URL could not be parsed: %s", u)
			continue
		}

		// Add scheme to URLs for convenience, so that you can write a command like: gemget example.com
		// instead of: gemget gemini://example.com
		if !strings.HasPrefix(u, "//") && !strings.Contains(u, "://") {
			u = "//" + u
			parsed, err = url.Parse(u)
			if err != nil {
				urlError("URL could not be parsed after adding scheme: %s", u)
				continue
			}
		}

		urls[i] = parsed
	}

	// Validate flags
	if len(flag.Args()) > 1 && *output != "" && *output != "-" {
		fatal("The output flag cannot be specified when there are multiple URLs, unless it is '-', meaning stdout.")
	}
	if *output == "-" {
		quiet = true
	}
	// Fetch each URL
	client := &gemini.Client{Insecure: *insecure}
	for _, u := range urls {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Info: Started %s\n", u)
		}
		fetch(u, client)
	}
}
