# utc2local-go

This command line tool converts UTC timestamps in stdin and write
converted lines to stdout.

Supported timestamp format:
  yyyy-mm-ddTHH:MM:SS{subsecond}{tz}
  where subsecond = (empty) | .SSS | .SSSSSS | .SSSSSSSSS
        tz = Z | +00:00

Usage:
```
$ ./utc2local -h
Usage of ./utc2local:
  -only-first
        convert only the first datetime in each line
  -tz string
        UTF timezone string to search (default "Z")
  -version
        show version and exit
```
