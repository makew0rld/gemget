package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/makeworld-the-better-one/gemget/scanner"
	"github.com/makeworld-the-better-one/go-gemini"
	flag "github.com/spf13/pflag"
)

var (
	version = "v1.8.0"
	commit  = "unknown"
	builtBy = "unknown"

	insecure       = flag.BoolP("insecure", "i", false, "Skip checking the cert\n")
	dir            = flag.StringP("directory", "d", ".", "\nThe directory where downloads go")
	output         = flag.StringP("output", "o", "", "Output path, for when there is only one URL.\n'-' means stdout and implies --quiet.\nIt overrides --directory.\n")
	errorSkip      = flag.BoolP("skip", "s", false, "Move to the next URL when one fails.")
	exts           = flag.BoolP("add-extension", "e", false, "Add .gmi extensions to gemini files that don't have it, like directories.\n")
	numRedirs      = flag.UintP("redirects", "r", 5, "How many redirects to follow before erroring out.")
	header         = flag.Bool("header", false, "\nPrint out (even with --quiet) the response header to stdout in the format:\nHeader: <status> <meta>\n")
	verFlag        = flag.BoolP("version", "v", false, "Find out what version of gemget you're running.")
	maxSize        = flag.StringP("max-size", "m", "", "Set the file size limit. Any download that exceeds this size will\ncause an Info output and be deleted.\nLeaving it blank or setting to zero bytes will result in no limit.\nThis flag is ignored when outputting to stdout.\nFormat: <num> <optional-byte-size>\nExamples: 423, 3.2KiB, '2.5 MB', '22 MiB', '10gib', 3M\n")
	maxSecs        = flag.UintP("max-time", "t", 0, "Set the downloading time limit, in seconds. Any download that\ntakes longer will cause an Info output and be deleted.\n")
	inputFilePath  = flag.StringP("input-file", "f", "", "Input file with a single URL on each line. Empty lines or lines starting\nwith # are ignored. URLs on the command line will be processed first.\n")
	noBar          = flag.Bool("no-progress-bar", false, "Disable the progress bar output.\n")
	proxy          = flag.StringP("proxy", "p", "", "A proxy that can requests are sent to instead.\nCan be a domain or IP with port. Port 1965 is assumed otherwise.\n")
	cert           = flag.String("cert", "", "Path to a PEM encoded TLS client certificate to be sent with the request.\n")
	key            = flag.String("key", "", "Path to a PEM encoded TLS key for the provided client cert.\n")
	connectTimeout = flag.Uint("connect-timeout", 15, "Max time allowed to form a connection and get the header, in seconds.\nSet to 0 for no timeout.")

	quiet bool // Set in main, so that it can be changed later if needed

	maxBytes  int64    // After maxSize is parsed this is set
	inputFile *os.File // Global var so it can be closed on fatal errors

	certPEM []byte
	keyPEM  []byte
)

func fatalIsFile(name, path string) {
	fi, err := os.Stat(path)
	if err == nil {
		if fi.IsDir() {
			fatal("%s path points to a directory: %s", name, path)
		}
	} else if os.IsNotExist(err) {
		fatal("%s does not exist: %v", path, err)
	} else {
		// Some other error - permissions for example
		fatal("Couldn't access %s at %s: %v", name, path, err)
	}
}

func main() {
	flag.BoolVarP(&quiet, "quiet", "q", false, "Neither info strings or the progress bar will be printed.\nNote that normally infos are printed to stderr, not stdout.\n")
	flag.Parse()

	if *verFlag {
		fmt.Println("gemget", version)
		fmt.Println("Commit:", commit)
		fmt.Println("Built by:", builtBy)
		return
	}

	// Validate urls
	if len(flag.Args()) == 0 && *inputFilePath == "" {
		flag.Usage()
		fmt.Println("\nNo URLs provided, by command line or file.")
		os.Exit(1)
	}

	// Validate flags
	if (len(flag.Args()) > 1 || *inputFilePath != "") && *output != "" && *output != "-" {
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
	if (*cert == "" && *key != "") || (*cert != "" && *key == "") {
		fatal("Both --cert and --key must be defined for a client certificate to be sent.")
	}

	// Validate cert and key files
	if *cert != "" && *key != "" {
		fatalIsFile("cert file", *cert)
		fatalIsFile("key file", *key)

		var err error
		certPEM, err = ioutil.ReadFile(*cert)
		if err != nil {
			fatal("Cert file could not be read: %v", err)
		}
		keyPEM, err = ioutil.ReadFile(*key)
		if err != nil {
			fatal("Key file could not be read: %v", err)
		}
	}

	// Validate output directory
	if *dir == "" {
		fatal("Directory path cannot be empty.")
	} else if *dir != "." {
		// Not the default, verify it's a real directory
		di, err := os.Stat(*dir)
		if err == nil {
			if !di.IsDir() {
				fatal("Directory path is a not a directory: %s", *dir)
			}
		} else if os.IsNotExist(err) {
			fatal("Directory does not exist: %s", *dir)
		} else {
			// Some other error - permissions for example
			fatal("Couldn't access directory %s: %v", *dir, err)
		}
	}

	// Validate input file
	if *inputFilePath != "" {
		fatalIsFile("input file", *inputFilePath)
	}

	client := &gemini.Client{
		Insecure:       *insecure,
		ConnectTimeout: time.Duration(*connectTimeout) * time.Second,
	}
	var urls *scanner.Scanner

	if *inputFilePath == "" {
		urls = scanner.NewScanner(nil, flag.Args()...)
	} else {
		var err error
		inputFile, err = os.Open(*inputFilePath)
		if err != nil {
			fatal("Issue opening input file: %v", err)
		}
		defer inputFile.Close()
		urls = scanner.NewScanner(inputFile, flag.Args()...)
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
