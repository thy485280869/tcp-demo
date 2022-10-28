package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	PACKET_EDGE_FLAG  = 0x123456
	PACKET_MAX_LENGTH = 1<<16 + 6 // PACKET_MAX_LENGTH = 2^16(max length of data) + 4(PACKET_EDGE_FLAG) + 2(data length)
)

var (
	ErrServerClosed = errors.New("tcp: Server closed")
)

func main() {
	closeDelay := true // close nagle?
	opt := TcpOption{CloseDelay: closeDelay}
	server := &TcpServer{Addr: ":8080", Opt: opt}
	if err := server.ListenAndServer(); err != nil {
		fmt.Println("server.ListenAndServer() err:", err)
	}
}

func handleConnection(c *net.TCPConn) {
	defer c.Close()
	fmt.Println(fmt.Sprintf("conn %s：", c.RemoteAddr().String()))

	result := bytes.NewBuffer(nil)
	var buf [PACKET_MAX_LENGTH]byte
	for {
		n, err := c.Read(buf[0:])
		result.Write(buf[0:n])
		if err != nil {
			if err == io.EOF {
				continue
			} else {
				fmt.Println("read err:", err)
				break
			}
		} else {
			scanner := bufio.NewScanner(result)
			scanner.Split(packetFunc)
			for scanner.Scan() {
				data := string(scanner.Bytes()[6:])
				fmt.Println("receive:", data)
				WriteData(c, data)
			}
		}
	}
}

func WriteData(c *net.TCPConn, data string) {
	res := strings.Split(data, "+")
	if len(res) == 2 {
		a, err := strconv.Atoi(res[0])
		b, err := strconv.Atoi(res[1])
		if err != nil {
			fmt.Println("send 0 (68)")
			c.Write(sendData([]byte("0")))
		} else {
			fmt.Printf("send %d (71)\n", a+b)
			c.Write(sendData([]byte(strconv.Itoa(a + b))))
		}
	} else {
		fmt.Println("send 0 (75)")
		c.Write(sendData([]byte("0")))
	}
}

func packetFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if !atEOF && len(data) > 6 && binary.BigEndian.Uint32(data[:4]) == PACKET_EDGE_FLAG {
		var l int16
		// read data(length in 0 ~ 2^16)
		binary.Read(bytes.NewReader(data[4:6]), binary.BigEndian, &l)
		pl := int(l) + 6
		if pl <= len(data) {
			return pl, data[:pl], nil
		}
	}
	return
}

func sendData(data []byte) []byte {
	magicNum := make([]byte, 4)
	binary.BigEndian.PutUint32(magicNum, 0x123456)
	lenNum := make([]byte, 2)
	binary.BigEndian.PutUint16(lenNum, uint16(len(data)))
	packetBuf := bytes.NewBuffer(magicNum)
	packetBuf.Write(lenNum)
	packetBuf.Write(data)
	return packetBuf.Bytes()
}

type TcpServer struct {
	Addr string
	Opt  TcpOption

	listener *net.TCPListener
	doneChan chan struct{} //send close msg to Serve
}

// TcpOption tpc-sever 配置
type TcpOption struct {
	WriteTimeout     time.Duration
	ReadTimeout      time.Duration
	KeepAliveTimeout time.Duration
	CloseDelay       bool
}

func (s *TcpServer) Close() {
	close(s.doneChan)
}

func (s *TcpServer) ListenAndServer() error {
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}

	addr := s.Addr
	if addr == "" {
		return errors.New("need addr")
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return errors.New("net.ResolveTCPAddr error")
	}
	tcpln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	return s.Serve(tcpln)
}

func (s *TcpServer) Serve(tcpln *net.TCPListener) error {
	s.listener = tcpln
	defer s.listener.Close()
	for {
		c, err := s.listener.AcceptTCP()
		if err != nil {
			select {
			case <-s.doneChan:
				return ErrServerClosed
			default:
			}
			fmt.Printf("accept fail, err: %v\n", err)
			continue
		}
		c = s.newConn(c)
		go handleConnection(c)
	}
	return nil
}

// newConn 根据s.opt配置conn
func (s *TcpServer) newConn(c *net.TCPConn) *net.TCPConn {
	if d := s.Opt.ReadTimeout; d != 0 {
		c.SetReadDeadline(time.Now().Add(d))
	}
	if d := s.Opt.WriteTimeout; d != 0 {
		c.SetWriteDeadline(time.Now().Add(d))
	}
	if d := s.Opt.KeepAliveTimeout; d != 0 {
		c.SetKeepAlive(true)
		c.SetKeepAlivePeriod(d)
	}
	if s.Opt.CloseDelay {
		c.SetNoDelay(true)
	}
	return c
}
