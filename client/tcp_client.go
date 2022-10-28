package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	PACKET_EDGE_FLAG  = 0x123456
	PACKET_MAX_LENGTH = 1<<16 + 6
)

func main() {

	target := "localhost:8080"
	raddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		log.Fatal(err)
	}
	c, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// test data
	dataArr := make([]string, 2)
	dataArr[0] = "1+2"
	dataArr[1] = "2-1"
	for i := 0; i < 2; i++ {
		// send data
		fmt.Println("send:", dataArr[i])
		data := []byte(dataArr[i])
		_, err = c.Write(sendData(data))
		if err != nil {
			log.Fatal(err)
		}

		// receive data
		result := bytes.NewBuffer(nil)
		var buf [PACKET_MAX_LENGTH]byte

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
			}
		}

	}
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

func packetFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if !atEOF && len(data) > 6 && binary.BigEndian.Uint32(data[:4]) == PACKET_EDGE_FLAG {
		var l int16
		binary.Read(bytes.NewReader(data[4:6]), binary.BigEndian, &l)
		pl := int(l) + 6
		if pl <= len(data) {
			return pl, data[:pl], nil
		}
	}
	return
}
