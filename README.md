# yamux-cli

## Install

```bash
brew install nwtgck/yamux-cli/yamux
```

Get more executables in the [releases](https://github.com/nwtgck/yamux-cli/releases) for you environment.

## Usage

```bash
... | yamux localhost 80 | ...
```

```bash
... | yamux -l 8080 | ...
```

Here is a complete example, but not useful. This is forwarding local 80 port to local 8080 port.

```bash
mkfifo my_pipe
cat my_pipe | yamux localhost 80 | yamux -l 8080 > ./my_pipe 
```

An expected usage of this CLI is to combine network tools and transport a remote port.
