package cmd

import (
	"net"
	"os"
	"time"
)

type stdioconn struct {}

func (_ *stdioconn) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (_ *stdioconn) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (_ *stdioconn) Close() error {
	return nil
}

func (_ *stdioconn) LocalAddr() net.Addr {
	return nil
}

func (_ *stdioconn) RemoteAddr() net.Addr {
	return nil
}

func (_ *stdioconn) SetDeadline(_ time.Time) error {
	return nil
}

func (_ *stdioconn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (_ *stdioconn) SetWriteDeadline(_ time.Time) error {
	return nil
}

