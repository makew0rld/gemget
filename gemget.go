package main

import (
	"fmt"
	"math"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/makeworld-the-better-one/gemget/scanner"
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
var maxSize = flag.StringP("max-size", "m", "", "Set the file size limit. Any download that exceeds this size will\ncause an Info output and be deleted.\nLeaving it blank or setting to zero bytes will result in no limit.\nThis flag is ignored when outputting to stdout.\nFormat: <num> <optional-byte-size>\nExamples: 423, 3.2KiB, '2.5 MB', '22 MiB', '10gib', 3M\n")
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
	if len(flag.Args()) == 0 && *inputFile == "" {
		flag.Usage()
		fmt.Println("\nNo URLs provided, by command line or file.")
		os.Exit(1)
	}
	// Validate flags
	if (len(flag.Args()) > 1 || *inputFile != "") && *output != "" && *output != "-" {
		fatal("The output flag cannot be specified when there are multiple URLs, unless it is '-', meaning stdout.")
	}
	if *maxSize != "" {
		tmpMaxBytes, err := humanize.ParseBytes(*maxSize)
		if err != nil {
			fatal("Max bytes string could not be parsed: %v", err)
		}
		if tmpMaxBytes > math.MaxInt64-1 {
			fatal("Max bytes is too large: %s,  %d bytes", *maxSize, tmpMaxBytes)
		}
		maxBytes = int64(tmpMaxBytes)
	}
	// Validate input file
	if *inputFile != "" {
		fi, err := os.Stat(*inputFile)
		if err == nil {
			if fi.IsDir() {
				fatal("Input file path points to a directory: %s", *inputFile)
			}
		} else {
			if os.IsNotExist(err) {
				fatal("Input file does not exist: %s", *inputFile)
			} else {
				// Some other error - permissions for example
				fatal("Couldn't access input file: %v", err)
			}
		}
	}

	client := &gemini.Client{Insecure: *insecure}
	var urls *scanner.Scanner

	if *inputFile == "" {
		urls = scanner.NewScanner(nil, flag.Args()...)
	} else {
		f, err := os.Open(*inputFile)
		if err != nil {
			fatal("Issue opening input file: %v", err)
		}
		defer f.Close()
		urls = scanner.NewScanner(f, flag.Args()...)
	}

	// Fetch each URL
	for urls.Scan() {
		if urls.Err() != nil {
			urlError("URL couldn't be parsed: %v", urls.Err())
		} else {
			info("Started %s", urls.URL().String())
			fetch(0, urls.URL(), client)
		}
	}
}
