package cmd

import (
	"errors"
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/nwtgck/yamux-cli/version"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

var flag struct {
	listens        bool
	usesUdp        bool
	usesUnixSocket bool
	showsVersion   bool
}

func init() {
	cobra.OnInitialize()
	RootCmd.PersistentFlags().BoolVarP(&flag.listens, "listen", "l", false, "listens")
	RootCmd.PersistentFlags().BoolVarP(&flag.showsVersion, "version", "", false, "show version")
	RootCmd.PersistentFlags().BoolVarP(&flag.usesUdp, "udp", "u", false, "UDP")
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
		if flag.usesUdp && flag.usesUnixSocket {
			return fmt.Errorf("unixgram not supported yet")
		}
		if flag.usesUdp {
			if flag.listens {
				host := "0.0.0.0"
				port := ""
				if len(args) == 2 {
					host = args[0]
					port = args[1]
				} else if len(args) == 1 {
					port = args[0]
				} else {
					return errors.New("port number is missing")
				}
				return udpYamuxClient(net.JoinHostPort(host, port))
			}
			if len(args) != 2 {
				return errors.New("host and port number are missing")
			}
			address := net.JoinHostPort(args[0], args[1])
			return udpYamuxServer(address)
		}
		if flag.listens {
			return handleTcpListen(args)
		}
		return handleTcpDial(args)
	},
}

func handleTcpListen(args []string) error {
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
	return tcpYamuxClient(ln)
}

func handleTcpDial(args []string) error {
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
	return tcpYamuxServer(dial)
}

func tcpYamuxServer(dial func() (net.Conn, error)) error {
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
			log.Printf("failed to dial: %v", err)
			continue
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

func tcpYamuxClient(ln net.Listener) error {
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
			log.Printf("failed to open: %v", err)
			continue
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

type udpAddrToYamuxStreamMap struct {
	inner *sync.Map
}

func (m udpAddrToYamuxStreamMap) Load(key *net.UDPAddr) *yamux.Stream {
	yamuxStream, ok := m.inner.Load(key.String())
	if !ok {
		return nil
	}
	return yamuxStream.(*yamux.Stream)
}

func (m udpAddrToYamuxStreamMap) Store(key *net.UDPAddr, value *yamux.Stream) {
	m.inner.Store(key.String(), value)
}

func udpYamuxClient(address string) error {
	raddrToYamuxStream := udpAddrToYamuxStreamMap{inner: new(sync.Map)}
	laddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	yamuxSession, err := yamux.Client(&stdioconn{}, nil)
	if err != nil {
		return err
	}
	var buf [65536]byte // NOTE: 1024 is not enough for vlc udp://@:5050

	for {
		n, raddr, err := conn.ReadFromUDP(buf[:])
		// TODO: remove or use flag to log after UDP feature is stable
		log.Printf("readfromudp: %s", raddr.String())
		if err != nil {
			return err
		}
		yamuxStream := raddrToYamuxStream.Load(raddr)
		if yamuxStream == nil {
			yamuxStream, err = yamuxSession.OpenStream()
			if err != nil {
				log.Printf("failed to open: %v", err)
				continue
			}
			// TODO: expire yamux stream
			raddrToYamuxStream.Store(raddr, yamuxStream)
			go func() {
				var buf [4096]byte
				for {
					n, err := yamuxStream.Read(buf[:])
					if err != nil {
						return
					}
					conn.WriteToUDP(buf[:n], raddr)
				}
			}()
		}

		go func() {
			yamuxStream.Write(buf[:n])
		}()
	}
}

func udpYamuxServer(address string) error {
	yamuxSession, err := yamux.Server(&stdioconn{}, nil)
	if err != nil {
		return err
	}
	for {
		yamuxStream, err := yamuxSession.AcceptStream()
		if err != nil {
			return err
		}
		conn, err := net.Dial("udp", address)
		if err != nil {
			log.Printf("failed to dial: %v", err)
			yamuxStream.Close()
			continue
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
