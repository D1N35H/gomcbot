package gomcbot

import (
	// "bufio"
	// "bytes"
	"fmt"
	// "net"

	"github.com/Tnze/gomcbot/network"
	pk "github.com/Tnze/gomcbot/network/packet"
)

//ProtocalVersion is the protocal version
// 477 for 1.14
const ProtocalVersion = 477

// PingAndList chack server status and list online player
// Return a JSON string about server status.
// see JSON format at https://wiki.vg/Server_List_Ping#Response
func PingAndList(addr string, port int) (string, error) {
	conn, err := network.DialMC(fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return "", err
	}

	//握手
	err = conn.WritePacket(
		//Handshake Packet
		pk.Marshal(
			0x00,                       //Handshake packet ID
			pk.VarInt(ProtocalVersion), //Protocal version
			pk.String(addr),            //Server's address
			pk.UnsignedShort(port),
			pk.Byte(1),
		))
	if err != nil {
		return "", fmt.Errorf("bot: send handshake packect fail: %v", err)
	}

	//请求服务器状态
	err = conn.WritePacket(pk.Marshal(0))
	if err != nil {
		return "", fmt.Errorf("bot: send list packect fail: %v", err)
	}

	//服务器返回状态
	recv, err := conn.ReadPacket()
	if err != nil {
		return "", fmt.Errorf("bot: recv list packect fail: %v", err)
	}
	var s pk.String
	err = recv.Scan(&s)
	return string(s), err
}

// JoinServer connect a Minecraft server for playing the game.
func (c *Client) JoinServer(addr string, port int) (err error) {
	//Connect
	c.conn, err = network.DialMC(fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		err = fmt.Errorf("bot: connect server fail: %v", err)
		return
	}

	//Handshake
	err = c.conn.WritePacket(
		//Handshake Packet
		pk.Marshal(
			0x00,                       //Handshake packet ID
			pk.VarInt(ProtocalVersion), //Protocal version
			pk.String(addr),            //Server's address
			pk.UnsignedShort(port),
			pk.Byte(2),
		))
	if err != nil {
		err = fmt.Errorf("bot: send handshake packect fail: %v", err)
		return
	}

	//Login
	err = c.conn.WritePacket(
		//LoginStart Packet
		pk.Marshal(0, pk.String(c.Name)))
	if err != nil {
		err = fmt.Errorf("bot: send login start packect fail: %v", err)
		return
	}

	for {
		//Recive Packet
		var pack pk.Packet
		pack, err = c.conn.ReadPacket()
		if err != nil {
			err = fmt.Errorf("bot: recv packet for Login fail: %v", err)
			return
		}

		//Handle Packet
		switch pack.ID {
		case 0x00: //Disconnect
			var reason pk.String
			err = pack.Scan(&reason)
			if err != nil {
				err = fmt.Errorf("bot: read Disconnect message fail: %v", err)
			} else {
				err = fmt.Errorf("bot: connect disconnected by server: %s", reason)
			}
			return
		case 0x01: //Encryption Request
			handleEncryptionRequest(c, pack)
		case 0x02: //Login Success
			// uuid, l := pk.UnpackString(pack.Data)
			// name, _ := unpackString(pack.Data[l:])
			return //switches the connection state to PLAY.
		case 0x03: //Set Compression
			var threshold pk.VarInt
			if err := pack.Scan(&threshold); err != nil {
				return fmt.Errorf("bot: set compression fail: %v", err)
			}
			c.conn.SetThreshold(int(threshold))
		case 0x04: //Login Plugin Request
			fmt.Println("Waring Login Plugin Request")
		}
	}
}
