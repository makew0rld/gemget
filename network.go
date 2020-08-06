package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/schollz/progressbar/v3"
)

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
		if *exts && !(strings.HasSuffix(name, ".gmi") || strings.HasSuffix(name, ".gemini")) && (resp.Meta == "" || strings.HasPrefix(resp.Meta, "text/gemini")) {
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

	if quiet || *noBar {
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
		if !quiet && !*noBar {
			fmt.Println()
		}
		if err == io.EOF {
			resp.Body.Close()
			info("Saved %s from URL %s", savePath, u.String())
			return
		}
		if err == nil {
			f.Close()
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
			f.Close()
			handleIOErr(err, resp, written, savePath, u.String())
			return
		}
	} else {
		// No size limit
		written, err = io.Copy(writer, resp.Body)
		if !quiet && !*noBar {
			fmt.Println()
		}
		if err != nil {
			f.Close()
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
	var resp *gemini.Response
	var err error
	if *proxy != "" {
		resp, err = client.FetchWithHost(*proxy, uStr)
	} else {
		resp, err = client.Fetch(uStr)
	}
	if err != nil {
		urlError(err.Error())
		return
	}
	defer resp.Body.Close()

	if *header {
		fmt.Printf("Header: %d %s\n", resp.Status, resp.Meta)
	}

	// Validate status
	switch gemini.SimplifyStatus(resp.Status) {
	case 60:
		urlError("%s needs a certificate, which is not implemented yet.", uStr)
		return
	case 30:
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
	case 10:
		urlError("This URL needs input, you should make the request again manually: %s", uStr)
	case 20:
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
	default:
		// Any sort of invalid status code will likely be caught by go-gemini, but this is here just in case
		urlError("URL returned status %d, skipping: %s", resp.Status, u)
	}
}
