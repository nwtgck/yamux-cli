package cmd

import (
	"errors"
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/nwtgck/yamux-cli/version"
	"github.com/spf13/cobra"
	"io"
	"net"
	"os"
)

var flag struct {
	listens        bool
	usesUnixSocket bool
	showsVersion   bool
}

func init() {
	cobra.OnInitialize()
	RootCmd.PersistentFlags().BoolVarP(&flag.listens, "listen", "l", false, "listens")
	RootCmd.PersistentFlags().BoolVarP(&flag.showsVersion, "version", "", false, "show version")
	// NOTE: long name 'unixsock' is from ncat (ref: https://manpages.debian.org/buster/ncat/nc.1.en.html)
	RootCmd.PersistentFlags().BoolVarP(&flag.usesUnixSocket, "unixsock", "U", false, "uses Unix-domain socket")
}

var RootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "yamux",
	Long:  "Multiplexer",
	Example: `
yamux localhost 80
yamux -l 8080
`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flag.showsVersion {
			fmt.Println(version.Version)
			return nil
		}
		if flag.listens {
			var ln net.Listener
			var err error
			if flag.usesUnixSocket {
				if len(args) != 1 {
					return errors.New("unix domain socket is missing")
				}
				ln, err = net.Listen("unix", args[0])
			} else {
				host := ""
				port := ""
				if len(args) == 2 {
					host = args[0]
					port = args[1]
				} else if len(args) == 1 {
					port = args[0]
				} else {
					return errors.New("port number is missing")
				}
				ln, err = net.Listen("tcp", net.JoinHostPort(host, port))
			}
			if err != nil {
				return err
			}
			return yamuxClient(ln)
		}
		var dial func() (net.Conn, error)
		if flag.usesUnixSocket {
			if len(args) != 1 {
				return errors.New("Unix-domain socket is missing")
			}
			dial = func() (net.Conn, error) {
				return net.Dial("unix", args[0])
			}
		} else {
			if len(args) != 2 {
				return errors.New("host and port number are missing")
			}
			address := net.JoinHostPort(args[0], args[1])
			dial = func() (net.Conn, error) {
				return net.Dial("tcp", address)
			}
		}
		return yamuxServer(dial)
	},
}

func yamuxServer(dial func() (net.Conn, error)) error {
	yamuxSession, err := yamux.Server(&stdioconn{}, nil)
	if err != nil {
		return err
	}
	for {
		yamuxStream, err := yamuxSession.Accept()
		if err != nil {
			return err
		}
		conn, err := dial()
		if err != nil {
			return err
		}
		fin := make(chan struct{})
		go func() {
			// TODO: hard code
			var buf = make([]byte, 4096)
			io.CopyBuffer(yamuxStream, conn, buf)
			fin <- struct{}{}
		}()
		go func() {
			// TODO: hard code
			var buf = make([]byte, 4096)
			io.CopyBuffer(conn, yamuxStream, buf)
			fin <- struct{}{}
		}()
		go func() {
			<-fin
			<-fin
			close(fin)
			conn.Close()
			yamuxStream.Close()
		}()
	}
}

func yamuxClient(ln net.Listener) error {
	yamuxSession, err := yamux.Client(&stdioconn{}, nil)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		yamuxStream, err := yamuxSession.Open()
		if err != nil {
			return err
		}
		fin := make(chan struct{})
		go func() {
			// TODO: hard code
			var buf = make([]byte, 4096)
			io.CopyBuffer(yamuxStream, conn, buf)
			fin <- struct{}{}
		}()
		go func() {
			// TODO: hard code
			var buf = make([]byte, 4096)
			io.CopyBuffer(conn, yamuxStream, buf)
			fin <- struct{}{}
		}()
		go func() {
			<-fin
			<-fin
			close(fin)
			conn.Close()
			yamuxStream.Close()
		}()
	}
}
