# gemget

A command line downloader for the [Gemini protocol](https://gemini.circumlunar.space/).
It works well with streams and can print headers for debugging as well.

```
gemget [option]... URL...

Usage of gemget:
  -e, --add-extension       Add .gmi extensions to gemini files that don't have it, like directories.
                            
  -d, --directory string    The directory where downloads go (default ".")
      --header              Print out (even with --quiet) the response header to stdout in the format:
                            Header: <status> <meta>
                            
  -f, --input-file string   Input file with a single URL on each line. Empty lines or lines starting
                            with # are ignored. URLs on the command line will be processed first.
                            
  -i, --insecure            Skip checking the cert
                            
  -m, --max-size string     Set the file size limit. Any download that exceeds this size will
                            cause an Info output and be deleted.
                            Leaving it blank or setting to zero bytes will result in no limit.
                            This flag is ignored when outputting to stdout.
                            Format: <num> <optional-byte-size>
                            Examples: 423, 3.2KiB, '2.5 MB', '22 MiB', '10gib', 3M
                            
  -t, --max-time uint       Set the downloading time limit, in seconds. Any download that
                            takes longer will cause an Info output and be deleted.
                            
      --no-progress-bar     Disable the progress bar output.
  -o, --output string       Output path, for when there is only one URL.
                            '-' means stdout and implies --quiet.
                            It overrides --directory.
                            
  -p, --proxy string        A proxy that can requests are sent to instead.
                            Can be a domain or IP with port. Port 1965 is assumed otherwise.
                            
  -q, --quiet               Neither info strings or the progress bar will be printed.
                            Note that normally infos are printed to stderr, not stdout.
                            
  -r, --redirects uint      How many redirects to follow before erroring out. (default 5)
  -s, --skip                Move to the next URL when one fails.
  -v, --version             Find out what version of gemget you're running.
```

# Installation
Install a binary of the most recent release from the [releases page](https://github.com/makeworld-the-better-one/gemget/releases/). On Unix-based systems you will have to make the file executable with `chmod +x <filename>`. You can rename the file to just `gemget` for easy access, and move it to `/usr/local/bin/`.

If you have Go installed, you can also install it using the Makefile.

```shell
git clone https://github.com/makeworld-the-better-one/gemget
cd gemget
# git checkout v1.2.3 # Optionally pin to a specific version instead of the latest commit
make
sudo make install
```

# Features to add
- Support client certificates
- Support interactive input for status code 10 & 11

## License
This project is under the [MIT License](./LICENSE).
