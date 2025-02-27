package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type TCPPeer struct {
	conn     net.Conn
	Outgoing bool
	ConnAddr string
}

func (p *TCPPeer) Send(b []byte) error {
	length := uint32(len(b))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)
	_, err := p.conn.Write(append(lengthBytes, b...))
	//_, err := p.conn.Write(b)
	return err
}

func (p *TCPPeer) readLoop(rpcCh chan RPC) {
	// buf := make([]byte, 4096) 这样写有问题，如果消息体大于4096，就会被截断，导致json解码失败
	for {
		// 读取长度前缀
		lengthBuf := make([]byte, 4)
		if _, err := io.ReadFull(p.conn, lengthBuf); err != nil {
			if err == io.EOF {
				continue
			}
			panic(err)
		}

		length := binary.BigEndian.Uint32(lengthBuf)
		msgBuf := make([]byte, length)
		_, err := io.ReadFull(p.conn, msgBuf)
		if err == io.EOF {
			continue
		}
		if err != nil {
			fmt.Printf("read error: %s", err)
			continue
		}

		rpcCh <- RPC{
			From:    p.conn.RemoteAddr(),
			Payload: bytes.NewReader(msgBuf),
		}
	}
}

type TCPTransport struct {
	peerCh     chan *TCPPeer
	listenAddr string
	listener   net.Listener
}

func NewTCPTransport(addr string, peerCh chan *TCPPeer) *TCPTransport {
	return &TCPTransport{
		peerCh:     peerCh,
		listenAddr: addr,
	}
}

func (t *TCPTransport) Start() error {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}

	t.listener = ln

	go t.acceptLoop()

	return nil
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("accept error from %+v\n", conn)
			continue
		}

		peer := &TCPPeer{
			conn:     conn,
			ConnAddr: conn.LocalAddr().String(),
		}

		// 注册自己的地址
		t.peerCh <- peer

		fmt.Printf("new incoming TCP connection => %+v\n", conn)

	}
}
