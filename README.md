# gemget

A command line downloader for the [Gemini protocol](https://gemini.circumlunar.space/).
It works well with streams and can print headers for debugging as well.

```
gemget [option]... URL...

Usage of gemget:
  -e, --add-extension      Add .gmi extensions to gemini files that don't have it, like directories.
                           
  -d, --directory string   The directory where downloads go (default ".")
      --header             Print out (even with --quiet) the response header to stdout in the format:
                           Header: <status> <meta>
                           
  -i, --insecure           Skip checking the cert
                           
  -m, --max-size string    Set the file size limit. Any download that exceeds this size will
                           cause an Error output and be deleted.
                           Leaving it blank or setting to zero bytes will result in no limit.
                           Format: <num> <optional-byte-size>
                           Examples: 423, 32 KiB, 20 MB, 22 MiB, 10 gib, 3M
                           
  -o, --output string      Output path, for when there is only one URL.
                           '-' means stdout and implies --quiet.
                           It overrides --directory.
                           
  -q, --quiet              No info strings will be printed. Note that normally infos are
                           printed to stderr, not stdout.
  -r, --redirects uint     How many redirects to follow before erroring out. (default 5)
  -s, --skip               Move to the next URL when one fails.
  -v, --version            Find out what version of gemget you're running.
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
Change the last part to say `gemget@master` to install from the latest commit

# Features to add
- Support TOFU with a certificate fingerprint cache, and option to disable it
- Support client certificates
- Support interactive input for status code 10 & 11
- Read URLs from file

## License
This project is under the [MIT License](./LICENSE).
