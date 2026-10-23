<p align="center"><img src="docs/logo.svg" width="96" alt="" /></p>

# cidrfold

cidrfold is a small command-line tool that merges, subtracts, intersects and queries lists of IPv4 CIDR blocks. It is written for people who maintain firewall allow-lists, security groups or ban lists. It uses only the Go standard library and runs on Windows, macOS and Linux.

[![CI](https://github.com/Gulliby-development/cidrfold/actions/workflows/ci.yml/badge.svg)](https://github.com/Gulliby-development/cidrfold/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Why

Allow-lists and ban lists grow by copy and paste until they hold overlapping and adjacent blocks. cidrfold folds them into the smallest equivalent list, subtracts one list from another, and tells you whether an address is covered, all from the command line and in scripts.

## Install

cidrfold needs Go 1.22 or newer.

```
go install github.com/Gulliby-development/cidrfold@latest
```

Or build from a checkout:

```
go build -o cidrfold .
```

## Commands

```
cidrfold merge [--format cidr|range|count] [FILE...]
cidrfold subtract [--format cidr|range|count] A B
cidrfold intersect [--format cidr|range|count] A B
cidrfold contains LIST ADDR...
```

`--format` is accepted as `--format VALUE` or `--format=VALUE` (default `cidr`). Options may stand anywhere after the command name, among the positional arguments too, as in `cidrfold merge a.txt b.txt --format range`; an option that takes a value uses the next argument whatever it looks like, and the last `--format` wins. A FILE, A, B or LIST that is `-` means standard input, which is read once. `merge` with no FILE reads standard input. `cidrfold --help` prints the usage.

### merge

Prints the minimal list of CIDR blocks covering the union of all entries, ascending.

Input lines `10.0.0.0/25`, `10.0.0.128/25`, `10.0.1.0/24`, `192.168.1.5`, `192.168.1.4` give:

```
10.0.0.0/23
192.168.1.4/31
```

With `--format range`:

```
10.0.0.0-10.0.1.255
192.168.1.4-192.168.1.5
```

Input lines `10.0.0.5/24`, `10.0.0.77`, `172.16.0.0/12`, `172.20.0.0/16` give:

```
10.0.0.0/24
172.16.0.0/12
```

Both ends of the address space work. The input lines `0.0.0.0/1` and `128.0.0.0/1` give:

```
0.0.0.0/0
```

The input lines `255.255.255.255` and `255.255.255.254` give:

```
255.255.255.254/31
```

### subtract

Prints the minimal CIDR cover of the addresses in A and not in B. `10.0.0.0/24` minus `10.0.0.64/26` gives:

```
10.0.0.0/26
10.0.0.128/25
```

With `--format range`:

```
10.0.0.0-10.0.0.63
10.0.0.128-10.0.0.255
```

An empty result prints nothing and exits 0.

### intersect

Prints the minimal CIDR cover of addresses present in both A and B. `10.0.0.0/23` intersected with `10.0.0.128/25`, `10.0.1.0/25` and `10.0.3.0/24` gives:

```
10.0.0.128/25
10.0.1.0/25
```

With `--format range`:

```
10.0.0.128-10.0.1.127
```

An empty result prints nothing and exits 0.

### contains

Reads LIST like `merge`, then reports each ADDR in argument order. With LIST `10.0.0.0/23` and the addresses `10.0.1.77 10.0.2.1`:

```
10.0.1.77 yes 10.0.0.0/23
10.0.2.1 no
```

## Counting addresses

Use `--format count` for the size of a result: `merge`, `subtract` and `intersect` then print one line with the number of addresses in the result, a decimal number from 0 to 4294967296, instead of the blocks.

With the five lines `10.0.0.0/25`, `10.0.0.128/25`, `10.0.1.0/24`, `192.168.1.5` and `192.168.1.4` in `a.txt`, `cidrfold merge --format count a.txt` prints:

```
514
```

With `10.0.0.0/24` in `c.txt` and `10.0.0.64/26` in `b.txt`, `cidrfold subtract --format=count c.txt b.txt` prints:

```
192
```

An empty result prints `0` (the other formats print nothing) and exits 0.

## Options

| Option | Description |
|---|---|
| `--format cidr` | Print CIDR blocks (default). Also `--format=cidr`. |
| `--format range` | Print `first-last` address ranges. Also `--format=range`. |
| `--format count` | Print the number of addresses in the result as one decimal number. Also `--format=count`. |
| `--help` | Print the usage. |

`--format` applies to `merge`, `subtract` and `intersect`.

## Input format

- IPv4 only, one entry per line; LF or CRLF line ends; a leading UTF-8 BOM is ignored.
- An entry is `a.b.c.d/n` (n from 0 to 32) or a bare `a.b.c.d` (same as `/32`). No leading zeros in octets or in n.
- Host bits may be set: `10.0.0.5/24` means `10.0.0.0/24`.
- Spaces and tabs around an entry are ignored. Blank lines and lines starting with `#` are ignored; a `#` after a space or tab starts a comment.
- Anything else is an input error, for example:

```
cidrfold: stdin:4: invalid entry "10.0.0.256"
```

The first error ends the run; nothing is printed on stdout.

## Exit codes

- 0: success (for `contains`, every address is covered).
- 1: `contains` found at least one address that is not covered.
- 2: any error: invalid input, invalid address, unknown command or option, wrong argument count, invalid `--format`, unreadable file.

## Development

```
go vet ./...
go test ./...
```

## License

MIT, see [LICENSE](LICENSE).

---

Built with BAKED, an open-source coding agent (bench loop round 7, BAKED 5c374a7).
