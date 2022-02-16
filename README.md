# yamux-cli
[![CI](https://github.com/nwtgck/yamux-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/nwtgck/yamux-cli/actions/workflows/ci.yml)

## Install for macOS

```bash
brew install nwtgck/yamux-cli/yamux
```

## Install for Ubuntu

```bash
wget https://github.com/nwtgck/yamux-cli/releases/download/v0.4.0/yamux-0.4.0-linux-amd64.deb
sudo dpkg -i yamux-0.4.0-linux-amd64.deb
```

Get more executables in the [releases](https://github.com/nwtgck/yamux-cli/releases) for you environment.

## Usage

### TCP

```bash
... | yamux localhost 80 | ...
```

```bash
... | yamux -l 8080 | ...
```

### Unix-domain socket

```bash
... | yamux -U /unix/domain/socket/path | ...
```

```bash
... | yamux -U -l /unix/domain/socket/path | ...
```

### UDP

```bash
... | yamux -u 1.1.1.1 53 | ...
```

```bash
... | yamux -ul 1053 | ...
```

## Complete example
Here is a complete example, but not useful. This is forwarding local 80 port to local 8080 port.

```bash
mkfifo my_pipe
cat my_pipe | yamux localhost 80 | yamux -l 8080 > ./my_pipe 
```

An expected usage of this CLI is to combine network tools and transport a remote port.
