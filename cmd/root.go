package cmd

import (
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/nwtgck/yamux-cli/version"
	"github.com/spf13/cobra"
	"io"
	"net"
	"os"
)

var listeningPort int
var showsVersion bool

func init() {
	cobra.OnInitialize()
	RootCmd.PersistentFlags().IntVarP(&listeningPort, "listen", "l", 0, "listening port")
	RootCmd.PersistentFlags().BoolVarP(&showsVersion, "version", "", false, "show version")
}

var RootCmd = &cobra.Command{
	Use:          os.Args[0],
	Short:        "yamux",
	Long:         "Multiplexer",
	Example: `
yamux localhost 80
yamux -l 8080
`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showsVersion {
			fmt.Println(version.Version)
			return nil
		}
		if len(args) == 2 {
			return yamuxServer(args[0], args[1])
		}
		if err := yamuxClient(); err != nil {
			return err
		}
		return nil
	},
}

func yamuxServer(host string, port string) error {
	yamuxSession, err := yamux.Server(&stdioconn{}, nil)
	if err != nil {
		return err
	}
	address := net.JoinHostPort(host, port)
	for {
		yamuxStream, err := yamuxSession.Accept()
		if err != nil {
			return err
		}
		conn, err := net.Dial("tcp", address)
		if err !=nil {
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

func yamuxClient() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", listeningPort))
	if err != nil {
		return err
	}
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
