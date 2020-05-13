# gemget

A command line downloader for the [Gemini protocol](https://gemini.circumlunar.space/).

```
gemget [option]... URL...

Usage of gemget:
  -e, --add-extension      Add .gmi extensions to gemini files that don't have it, like directories.
  -d, --directory string   The directory where downloads go (default ".")
  -i, --insecure           Skip checking the cert
  -o, --output string      Output file, for when there is only one URL.
                           '-' means stdout and implies --quiet.
  -q, --quiet              No output except for errors.
  -r, --redirects uint     How many redirects to follow before erroring out. (default 5)
  -s, --skip               Move to the next URL when one fails.
```

# Installation
```
curl -sf https://gobinaries.com/makeworld-the-better-one/gemget | sh
```
Or install a binary of the most recent release from the [releases page](https://github.com/makeworld-the-better-one/gemget/releases/).

If you have Go installed, you can also install it with:
```
GO111MODULE=on go get -u github.com/makeworld-the-better-one/gemget
```

If you want to install from the latest commit (not release), clone the repo and then run `go install` inside it.

# Features to add
- Support TOFU with a certificate fingerprint cache, and option to disable it
- Support client certificates
  - This requires forking the [go-gemini](https://git.sr.ht/~yotam/go-gemini) library this project uses, as it doesn't support that
- Support self-signed certs
  - Using `--insecure` can get around this, but this also disables checking for expiry dates, etc.
- Support interactive input for status code 10
- Read URLs from file

## License
This project is under the [MIT License](./LICENSE).
