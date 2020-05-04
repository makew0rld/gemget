# gemget

A command line downloader for the [Gemini protocol](https://gemini.circumlunar.space/).

```
gemget [option]... URL...

Usage of gemget:
  -e, --add-extension     Add .gmi extensions to gemini files that don't have it, like directories.
  -d, --directory string   The directory where downloads go (default ".")
      --follow             Follow redirects, up to 5. (default true)
      --insecure           Skip checking the cert
  -o, --output string      Output file, for when there is only one URL.
                           '-' means stdout.
  -q, --quiet              No output except for errors.
      --skip               Move to the next URL when one fails. (default true)
```

# Features to add
- Support TOFU with a certificate fingerprint cache, and option to disable it
- Support client certificates
  - This requires forking the [go-gemini](https://git.sr.ht/~yotam/go-gemini) library this project uses, as it doesn't support that
- Support interactive input for status code 10
- Read URLs from file

## License
This project is under the [MIT License](./LICENSE).
