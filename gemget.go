package main

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/makeworld-the-better-one/go-gemini"
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
var inputFile = flag.StringP("input-file", "f", "", "Input file with a single URL on each line. Empty lines or lines starting\nwith # are ignored. URLs on the command line will be processed first.\n")

var maxBytes int64 // After maxSize is parsed this is set

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
