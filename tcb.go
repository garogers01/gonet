package main

import (
	"errors"
	"fmt"
	"golang.org/x/net/ipv4"
	"time"
)

type TCB struct {
	read           chan *TCP_Packet
	writer         *ipv4.RawConn
	ipAddress      string      // destination ip address
	srcIP          string      // src ip address
	lport, rport   uint16      // ports
	seqNum, ackNum uint32      // sequence number
	state          uint        // from the FSM
	kind           uint        // type (server or client)
	serverParent   *Server_TCB // the parent server
	curWindow      uint16      // the current window size
	sendBuffer     []byte      // a buffer of bytes that need to be sent
	urgSendBuffer  []byte      // buffer of urgent data TODO urg data later
	recvBuffer     []byte      // bytes to pass to the application above
}

func New_TCB(local, remote uint16, dstIP string, read chan *TCP_Packet, write *ipv4.RawConn, kind uint) (*TCB, error) {
	fmt.Println("New_TCB")
	c := &TCB{
		lport:        local,
		rport:        remote,
		ipAddress:    dstIP,
		srcIP:        "127.0.0.1", // TODO: don't hardcode the srcIP
		read:         read,
		writer:       write,
		seqNum:       genRandSeqNum(), // TODO verify that this works
		ackNum:       uint32(0),       // Always 0 at start
		state:        CLOSED,
		kind:         kind,
		serverParent: nil,
		curWindow:    43690, // TODO calc using http://ithitman.blogspot.com/2013/02/understanding-tcp-window-window-scaling.html
	}
	fmt.Println("Starting the packet dealer")

	go c.PacketSender()
	go c.PacketDealer()

	return c, nil
}

func (c *TCB) UpdateState(newState uint) {
	c.state = newState
	// TODO notify of the update
}

func (c *TCB) PacketSender() {
	// TODO: deal with data in send and urgSend buffers
}

func (c *TCB) PacketDealer() {
	// read each tcp packet and deal with it
	fmt.Println("Packet Dealing")
	for {
		fmt.Println("Waiting for packets")
		segment := <-c.read
		fmt.Println("got a packet")
		// TODO check the reset flag first
		switch c.state {
		case CLOSED:
			go c.DealClosed(segment)
		case SYN_SENT:
			go c.DealSynSent(segment)
		case SYN_RCVD:
			go c.DealSynRcvd(segment)
		case ESTABLISHED:
			go c.DealEstablished(segment)
			// TODO fill other possible states
		}
	}
}

func (c *TCB) DealClosed(d *TCP_Packet) {
	// TODO: send reset
}

func (c *TCB) DealSynSent(d *TCP_Packet) {
	//fmt.Println("in state syn-sent")
	if d.header.flags&TCP_SYN != 0 && d.header.flags&TCP_ACK != 0 {
		// received SYN-ACK
		fmt.Println("Recieved syn-ack")

		// TODO: verify the seq and ack fields

		// Send ACK
		c.seqNum++ // A+1
		B := d.header.seq
		c.ackNum = B + 1
		ACK, err := (&TCP_Header{
			srcport: c.lport,
			dstport: c.rport,
			seq:     c.seqNum,
			ack:     c.ackNum,
			flags:   TCP_ACK,
			window:  c.curWindow, // TODO improve the window field calculation
			urg:     0,
			options: []byte{},
		}).Marshal_TCP_Header(c.ipAddress, c.srcIP)
		if err != nil {
			fmt.Println(err) // TODO log not print
			return
		}

		err = MyRawConnTCPWrite(c.writer, ACK, c.ipAddress)
		fmt.Println("Sent ACK data")
		if err != nil {
			fmt.Println(err) // TODO log not print
			return
		}
		c.UpdateState(ESTABLISHED)
	} else if d.header.flags&TCP_SYN == 0 {
		// TODO deal with special case: http://www.tcpipguide.com/free/t_TCPConnectionEstablishmentProcessTheThreeWayHandsh-4.htm (Simultaneous Open Connection Establishment)
	} else {
		// drop otherwise
	}
}

func (c *TCB) DealSynRcvd(d *TCP_Packet) {
	if d.header.flags&TCP_SYN != 0 {
		// TODO send reset
	}
}

func (c *TCB) DealEstablished(d *TCP_Packet) {
	// TODO deal with data
}

func (c *TCB) Connect() error {
	if c.kind != TCP_CLIENT || c.state != CLOSED {
		return errors.New("TCB is not a closed client")
	}
	// Send the SYN packet
	SYN, err := (&TCP_Header{
		srcport: c.lport,
		dstport: c.rport,
		seq:     c.seqNum,
		ack:     c.ackNum,
		flags:   TCP_SYN,
		window:  c.curWindow, // TODO improve the window size calculation
		urg:     0,
		options: []byte{0x02, 0x04, 0xff, 0xd7, 0x04, 0x02, 0x08, 0x0a, 0x02, 0x64, 0x80, 0x8b, 0x0, 0x0, 0x0, 0x0, 0x01, 0x03, 0x03, 0x07}, // TODO compute the options of SYN instead of hardcoding them
	}).Marshal_TCP_Header(c.ipAddress, c.srcIP)
	if err != nil {
		return err
	}

	//c.writer.WriteTo(SYN)
	err = MyRawConnTCPWrite(c.writer, SYN, c.ipAddress)
	fmt.Println("Sent SYN")
	if err != nil {
		return err
	}
	c.UpdateState(SYN_SENT)

	// TODO set up resend SYN timers

	// wait for the connection state to be ready
	// TODO use sync.Cond broadcast to avoid the infinite for loop
	for {
		st := c.state
		if st == CLOSED {
			return errors.New("Connection timed out and closed, or reset.")
		} else if st == ESTABLISHED {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func (c *TCB) Send(data []byte) error { // a non-blocking send call
	c.sendBuffer = append(c.sendBuffer, data...)
	return nil // TODO: read and send from the send buffer
}

func (c *TCB) Recv(num uint64) ([]byte, error) {
	return nil, nil // TODO: implement TCB receive
}

func (c *TCB) Close() error {
	return nil // TODO: free manager read buffer and send fin/fin+ack/etc. Also kill timers with a wait group
}

// TODO: support a status call

func (c *TCB) Abort() error {
	// TODO: kill all timers
	// TODO: kill all long term processes
	// TODO: send a reset
	// TODO: delete the TCB + assoc. data
	return nil
}
