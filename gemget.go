package main

import (
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/schollz/progressbar/v3"
	flag "github.com/spf13/pflag"
)

var version = "1.3.0-unreleased"

var insecure = flag.BoolP("insecure", "i", false, "Skip checking the cert\n")
var dir = flag.StringP("directory", "d", ".", "The directory where downloads go")
var output = flag.StringP("output", "o", "", "Output path, for when there is only one URL.\n'-' means stdout and implies --quiet.\nIt overrides --directory.\n")
var errorSkip = flag.BoolP("skip", "s", false, "Move to the next URL when one fails.")
var exts = flag.BoolP("add-extension", "e", false, "Add .gmi extensions to gemini files that don't have it, like directories.\n")
var quiet bool // Set in main, so that it can be changed later if needed
var numRedirs = flag.UintP("redirects", "r", 5, "How many redirects to follow before erroring out.")
var header = flag.Bool("header", false, "Print out (even with --quiet) the response header to stdout in the format:\nHeader: <status> <meta>\n")
var verFlag = flag.BoolP("version", "v", false, "Find out what version of gemget you're running.")
var maxSize = flag.StringP("max-size", "m", "", "Set the file size limit. Any download that exceeds this size will\ncause an Info output and be deleted.\nLeaving it blank or setting to zero bytes will result in no limit.\nThis flag is ignored when outputting to stdout.\nFormat: <num> <optional-byte-size>\nExamples: 423, 32 KiB, 20 MB, 22 MiB, 10 gib, 3M\n")
var maxSecs = flag.UintP("max-time", "t", 0, "Set the downloading time limit, in seconds. Any download that\ntakes longer will cause an Info output and be deleted.\n")

var maxBytes int64 // After maxSize is parsed this is set

func fatal(format string, a ...interface{}) {
	urlError(format, a...)
	os.Exit(1)
}

func urlError(format string, a ...interface{}) {
	format = "Error: " + strings.TrimRight(format, "\n") + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
	if !*errorSkip {
		os.Exit(1)
	}
}

func info(format string, a ...interface{}) {
	if quiet {
		return
	}
	format = "Info: " + strings.TrimRight(format, "\n") + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}

func handleIOErr(err error, resp *gemini.Response, written int64, path, u string) {
	if *maxSecs > 0 && strings.HasSuffix(err.Error(), "use of closed network connection") {
		// Download timed out intentionally, due to a user flag

		if path != "" && path != "-" {
			// A real file path is being downloaded to

			err = os.Remove(path)
			if err != nil {
				fatal("Tried to remove %s (from URL %s) because the download timed out, but encountered this error: %v", path, u, err)
			}
			info("Download timed out, deleted: %s", u)
		} else {
			// Download is going to stdout
			info("Download timed out: %s", u)
		}
		return
	}
	resp.Body.Close()
	fatal("Issue saving file %s, %d bytes saved: %v", path, written, err)
}

func saveFile(resp *gemini.Response, u *url.URL) {
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
		resp.Body.Close()
		fatal("Couldn't create file %s: %v", savePath, err)
	}
	defer f.Close()

	var writer io.Writer

	if quiet {
		writer = f // Just write to file only
	} else {
		bar := progressbar.DefaultBytes(-1, "downloading")
		writer = io.MultiWriter(f, bar) // Use progress bar as well
	}

	var written int64

	if maxBytes > 0 {
		// Try to read one byte more than the limit. If EOF is returned, then the response
		// was at the limit or below. Otherwise it was too large.
		written, err = io.CopyN(writer, resp.Body, maxBytes+1)
		if !quiet {
			fmt.Println()
		}
		if err == io.EOF {
			resp.Body.Close()
			info("Saved %s from URL %s", savePath, u.String())
			return
		}
		if err == nil {
			err = os.Remove(savePath)
			if err != nil {
				resp.Body.Close()
				fatal("Tried to remove %s (from URL %s) because it was larger than the max size limit, but encountered this error: %v", savePath, u.String(), err)
			}
			resp.Body.Close()
			info("File is larger than max size limit, deleted: %s", u.String())
			return
		} else if err != io.EOF {
			// Some other error
			handleIOErr(err, resp, written, savePath, u.String())
			return
		}
	} else {
		// No size limit
		written, err = io.Copy(writer, resp.Body)
		if !quiet {
			fmt.Println()
		}
		if err != nil {
			handleIOErr(err, resp, written, savePath, u.String())
		} else {
			resp.Body.Close()
			info("Saved %s from URL %s", savePath, u.String())
		}
	}
}

// fetch fetches the URL.
// n is how many redirects have happened. Set to 0 for the first request.
func fetch(n uint, u *url.URL, client *gemini.Client) {
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
		if n >= *numRedirs {
			urlError("URL redirected too many times: %s", uStr)
			return
		}
		// Follow the recursion
		redirect, err := url.Parse(resp.Meta)
		if err != nil {
			urlError("Redirect URL %s couldn't be parsed.", resp.Meta)
			return
		}
		info("Redirected to %s", resp.Meta)
		fetch(n+1, u.ResolveReference(redirect), client)
	} else if gemini.SimplifyStatus(resp.Status) == 10 {
		urlError("This URL needs input, you should make the request again manually: %s", uStr)
	} else if gemini.SimplifyStatus(resp.Status) == 20 {

		if *maxSecs > 0 {
			// Goroutine that closes response after timeout
			go func(r *gemini.Response) {
				time.Sleep(time.Duration(*maxSecs) * time.Second)
				r.Body.Close()
			}(resp)
		}

		// Output to stdout, otherwise save it to a file
		if *output == "-" {
			written, err := io.Copy(os.Stdout, resp.Body)
			if err != nil {
				handleIOErr(err, resp, written, "", u.String())
			}
			return
		}
		saveFile(resp, u)
		return
	} else {
		// Any sort of invalid status code will likely be caught by go-gemini, but this is here just in case
		urlError("URL returned status %d, skipping: %s", resp.Status, u)
	}
}

func main() {
	flag.BoolVarP(&quiet, "quiet", "q", false, "No info strings will be printed. Note that normally infos are\nprinted to stderr, not stdout.")
	flag.Parse()

	if *verFlag {
		fmt.Println("gemget " + version)
		return
	}

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
			u = "gemini://" + u
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

	if *maxSize != "" {
		tmpMaxBytes, err := humanize.ParseBytes(*maxSize)
		if err != nil {
			fatal("Max bytes string could not be parsed: %v", err)
		}
		if tmpMaxBytes > math.MaxInt64-1 {
			fatal("Max bytes is too large: %s = %d bytes", *maxSize, tmpMaxBytes)
		}
		maxBytes = int64(tmpMaxBytes)
	}

	// Fetch each URL
	client := &gemini.Client{Insecure: *insecure}
	for _, u := range urls {
		info("Started %s", u)
		fetch(0, u, client)
	}
}
