# gosmi

## Fork Status

For now, don't use this fork, it's experimental. I've renamed the interal references from "github.com/sleepinggenius2/gosmi" --> "github.com/lukeod/gosmi" purely to simplify testing tooling that compare the results of parsing a mib library between the current main "github.com/sleepinggenius2/gosmi" and this fork.

## Summary

Starting with v0.2.0, this library is native Go and no longer a wrapper around libsmi. The implementation is currently very close, but may change in the future.

For the native implementation, two additional components have been added:

* SMIv1/2 parser in [parser](parser)
* libsmi-compatible Go implementation in [smi](smi)

## Usage

On Ubuntu for v0.1.0 and below: `$ sudo apt-get install libsmi2-dev`

### Examples

Examples can now be found in:

* [cmd/parse](cmd/parse)
* [cmd/smi](cmd/smi)
* [cmd/embed](cmd/embed)
