package gomcbot

import (
	// "bytes"
	"fmt"
	"github.com/Tnze/gomcbot/data"
	pk "github.com/Tnze/gomcbot/network/packet"
	// "math"
	// "time"
)

// //GetPosition return the player's position
// func (p *Player) GetPosition() (x, y, z float64) {
// 	return p.X, p.Y, p.Z
// }

// //GetBlockPos return the position of the Block at player's feet
// func (p *Player) GetBlockPos() (x, y, z int) {
// 	return int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z))
// }

// //PlayerInfo content player info in server.
// type PlayerInfo struct {
// 	EntityID         int      //实体ID
// 	Gamemode         int      //游戏模式
// 	Hardcore         bool     //是否是极限模式
// 	Dimension        int      //维度
// 	Difficulty       int      //难度
// 	LevelType        string   //地图类型
// 	ReducedDebugInfo bool     //减少调试信息
// 	SpawnPosition    Position //主世界出生点
// }

// // PlayerAbilities defines what player can do.
// type PlayerAbilities struct {
// 	Flags               int8
// 	FlyingSpeed         float32
// 	FieldofViewModifier float32
// }

// //Position is a 3D vector.
// type Position struct {
// 	X, Y, Z int
// }

// HandleGame recive server packet and response them correctly.
// Note that HandleGame will block if you don't recive from Events.
func (c *Client) HandleGame() error {
	for {
		//Read packets
		p, err := c.conn.ReadPacket()
		if err != nil {
			return fmt.Errorf("bot: read packet fail: %v", err)
		}
		//handle packets
		err = c.handlePacket(p)
		if err != nil {
			return fmt.Errorf("handle packet 0x%X error: %v", p.ID, err)
		}
	}
}

func (c *Client) handlePacket(p pk.Packet) (err error) {
	switch p.ID {
	case data.JoinGame:
		err = handleJoinGamePacket(c, p)
	case data.PluginMessageClientbound:
		err = handlePluginPacket(c, p)
	case data.ServerDifficulty:
		err = handleServerDifficultyPacket(c, p)
	case data.SpawnPosition:
		err = handleSpawnPositionPacket(c, p)
	case data.PlayerAbilitiesClientbound:
		err = handlePlayerAbilitiesPacket(c, p)
		c.conn.WritePacket(
			//ClientSettings packet (serverbound)
			pk.Marshal(
				data.ClientSettings,
				pk.String(c.settings.Locale),
				pk.Byte(c.settings.ViewDistance),
				pk.VarInt(c.settings.ChatMode),
				pk.Boolean(c.settings.ChatColors),
				pk.UnsignedByte(c.settings.DisplayedSkinParts),
				pk.VarInt(c.settings.MainHand),
			),
		)
	case data.HeldItemChangeClientbound:
		err = handleHeldItemPacket(c, p)
	case data.ChunkData:
		err = handleChunkDataPacket(c, p)
	case data.PlayerPositionAndLookClientbound:
		err = handlePlayerPositionAndLookPacket(c, p)
		sendPlayerPositionAndLookPacket(c) // to confirm the position
	case 0x5A:
		// handleDeclareRecipesPacket(g, reader)
	case 0x29:
		// err = handleEntityLookAndRelativeMove(g, reader)
	case 0x3B:
		// handleEntityHeadLookPacket(g, reader)
	case 0x28:
		// err = handleEntityRelativeMovePacket(g, reader)
	case data.KeepAliveClientbound:
		err = handleKeepAlivePacket(c, p)
	case 0x2B:
		//handleEntityPacket(g, reader)
	case 0x05:
		// err = handleSpawnPlayerPacket(g, reader)
	case data.WindowItems:
		err = handleWindowItemsPacket(c, p)
	case data.UpdateHealth:
		err = handleUpdateHealthPacket(c, p)
	case data.ChatMessageClientbound:
		err = handleChatMessagePacket(c, p)
	case data.BlockChange:
		err = handleBlockChangePacket(c, p)
	case data.MultiBlockChange:
		err = handleMultiBlockChangePacket(c, p)
	case 0x1A:
		// should assumes that the server has already closed the connection by the time the packet arrives.

		err = fmt.Errorf("disconnect")
	case 0x16:
	// 	err = handleSetSlotPacket(g, reader)
	case 0x51:
		// err = handleSoundEffect(g, reader)
	default:
		// fmt.Printf("ignore pack id %X\n", p.ID)
	}
	return
}

// func handleSoundEffect(g *Client, r *bytes.Reader) error {
// 	SoundID, err := pk.UnpackVarInt(r)
// 	if err != nil {
// 		return err
// 	}
// 	SoundCategory, err := pk.UnpackVarInt(r)
// 	if err != nil {
// 		return err
// 	}

// 	x, err := pk.UnpackInt32(r)
// 	if err != nil {
// 		return err
// 	}
// 	y, err := pk.UnpackInt32(r)
// 	if err != nil {
// 		return err
// 	}
// 	z, err := pk.UnpackInt32(r)
// 	if err != nil {
// 		return err
// 	}
// 	Volume, err := pk.UnpackFloat(r)
// 	if err != nil {
// 		return err
// 	}
// 	Pitch, err := pk.UnpackFloat(r)
// 	if err != nil {
// 		return err
// 	}
// 	g.events <- SoundEffectEvent{SoundID, SoundCategory, float64(x) / 8, float64(y) / 8, float64(z) / 8, Volume, Pitch}

// 	return nil
// }

// func handleSetSlotPacket(g *Client, r *bytes.Reader) error {
// 	windowID, err := r.ReadByte()
// 	if err != nil {
// 		return err
// 	}
// 	slot, err := pk.UnpackInt16(r)
// 	if err != nil {
// 		return err
// 	}
// 	slotData, err := unpackSolt(r)
// 	if err != nil {
// 		return err
// 	}

// 	switch int8(windowID) {
// 	case 0:
// 		if slot < 32 || slot > 44 {
// 			// return fmt.Errorf("slot out of range")
// 			break
// 		}
// 		fallthrough
// 	case -2:
// 		g.player.Inventory[slot] = slotData
// 		g.events <- InventoryChangeEvent(slot)
// 	}
// 	return nil
// }

func handleMultiBlockChangePacket(c *Client, p pk.Packet) error {
	if !c.settings.ReciveMap {
		return nil
	}

	var cX, cY pk.Int

	err := p.Scan(&cX, &cY)
	if err != nil {
		return err
	}

	c := g.wd.chunks[chunkLoc{int(cX), int(cY)}]
	if c != nil {
		RecordCount, err := pk.UnpackVarInt(r)
		if err != nil {
			return err
		}

		for i := int32(0); i < RecordCount; i++ {
			xz, err := r.ReadByte()
			if err != nil {
				return err
			}
			y, err := r.ReadByte()
			if err != nil {
				return err
			}
			BlockID, err := pk.UnpackVarInt(r)
			if err != nil {
				return err
			}
			x, z := xz>>4, xz&0x0F

			c.sections[y/16].blocks[x][y%16][z] = Block{id: uint(BlockID)}
		}
	}

	return nil
}

func handleBlockChangePacket(c *Client, p pk.Packet) error {
	if !c.settings.ReciveMap {
		return nil
	}
	var pos pk.Position
	err := p.Scan(&pos)
	if err != nil {
		return err
	}

	c := g.wd.chunks[chunkLoc{x >> 4, z >> 4}]
	if c != nil {
		id, err := pk.UnpackVarInt(r)
		if err != nil {
			return err
		}
		c.sections[y/16].blocks[x&15][y&15][z&15] = Block{id: uint(id)}
	}

	return nil
}

func handleChatMessagePacket(c *Client, p pk.Packet) error {
	var (
		s   pk.String
		pos pk.Byte
	)

	if err := p.Scan(&s, &pos); err != nil {
		return err
	}

	cm, err := newChatMsg([]byte(s))
	if err != nil {
		return err
	}

	return nil
}

func handleUpdateHealthPacket(c *Client, p pk.Packet) (err error) {
	var (
		Health         pk.Float
		Food           pk.VarInt
		FoodSaturation pk.Float
	)

	err = p.Scan(&Health, &Food, &FoodSaturation)
	if err != nil {
		return
	}

	c.player.Health = Health
	c.player.Food = Food
	c.player.FoodSaturation = FoodSaturation

	if c.player.Health < 1 { //player is dead
		sendPlayerPositionAndLookPacket(c)
		time.Sleep(time.Second * 2)  //wait for 2 sec make it more like a human
		sendClientStatusPacket(c, 0) //status 0 means perform respawn
	}
	return
}

func handleJoinGamePacket(g *Client, p pk.Packet) error {
	var (
		eid        pk.Int
		gamemode   pk.Byte
		dimension  pk.Int
		difficulty pk.Byte
		maxPlayers pk.Byte
		levelType  pk.String
		rdi        pk.Byte
	)
	err := p.Scan(&eid, &gamemode, &dimension, &difficulty, &maxPlayers, &levelType, &rdi)
	if err != nil {
		return err
	}

	g.Info.EntityID = int(eid)
	g.Info.Gamemode = int(gamemode & 0x7)
	g.Info.Hardcore = gamemode&0x8 != 0
	g.Info.Dimension = int(dimension)
	g.Info.Difficulty = int(difficulty)
	g.Info.LevelType = strinig(levelType)
	g.Info.ReducedDebugInfo = byte(rdi) != 0x00
	return nil
}

func handlePluginPacket(c *Client, p pk.Packet) error {
	// fmt.Println("Plugin Packet: ", p)
	return nil
}

func handleServerDifficultyPacket(c *Client, p pk.Packet) error {
	var difficulty pk.Byte
	err = p.Scan(&difficulty)
	if err != nil {
		return err
	}
	g.Info.Difficulty = int(difficulty)
	return nil
}

func handleSpawnPositionPacket(c *Client, p pk.Packet) error {
	var pos pk.Position
	err = p.Scan(&pos)
	if err != nil {
		return err
	}
	c.Info.SpawnPosition.X, c.Info.SpawnPosition.Y, c.Info.SpawnPosition.Z =
		pos.X, pos.Y, pos.Z
	return nil
}

func handlePlayerAbilitiesPacket(g *Client, p pk.Packet) error {
	var (
		flags    pk.Byte
		flySpeed pk.Float
		viewMod  pk.Float
	)
	err := p.Scan(&flags, &flySpeed, &viewMod)
	if err != nil {
		return err
	}
	g.abilities.Flags = int8(flags)
	g.abilities.FlyingSpeed = float32(flySpeed)
	g.abilities.FieldofViewModifier = float32(viewMod)
	return nil
}

func handleHeldItemPacket(c *Client, p pk.Packet) error {
	var hi pk.Byte
	if err := p.Scan(&hi); err != nil {
		return err
	}
	g.player.HeldItem = int(hi)
	return nil
}

func handleChunkDataPacket(g *Client, p pk.Packet) error {
	if !g.settings.ReciveMap {
		return nil
	}

	c, x, y, err := unpackChunkDataPacket(p, g.Info.Dimension == 0)
	g.wd.chunks[chunkLoc{x, y}] = c
	return err
}

// var isSpawn bool

func handlePlayerPositionAndLookPacket(c *Client, p pk.Packet) error {
	var (
		x, y, z    pk.Double
		yaw, pitch pk.Float
		flags      pk.Byte
		TeleportID pk.VarInt
	)

	err := p.Scan(&x, &y, &z, &yaw, &pitch, &flags, &TeleportID)
	if err != nil {
		return err
	}

	if flags&0x01 == 0 {
		c.player.X = x
	} else {
		c.player.X += x
	}
	if flags&0x02 == 0 {
		c.player.Y = y
	} else {
		c.player.Y += y
	}
	if flags&0x04 == 0 {
		c.player.Z = z
	} else {
		c.player.Z += z
	}
	if flags&0x08 == 0 {
		c.player.Yaw = yaw
	} else {
		c.player.Yaw += yaw
	}
	if flags&0x10 == 0 {
		c.player.Pitch = pitch
	} else {
		c.player.Pitch += pitch
	}
	sendTeleportConfirmPacket(c, TeleportID)

	return nil
}

// func handleDeclareRecipesPacket(g *Client, r *bytes.Reader) {
// 	//Ignore Declare Recipes Packet

// 	// NumRecipes, index := pk.UnpackVarInt(p.Data)
// 	// for i := 0; i < int(NumRecipes); i++ {
// 	// 	RecipeID, len := pk.UnpackString(p.Data[index:])
// 	// 	index += len
// 	// 	Type, len := pk.UnpackString(p.Data[index:])
// 	// 	index += len
// 	// 	switch Type {
// 	// 	case "crafting_shapeless":
// 	// 	}
// 	// }
// }

// func handleEntityLookAndRelativeMove(g *Client, r *bytes.Reader) error {
// 	ID, err := pk.UnpackVarInt(r)
// 	if err != nil {
// 		return err
// 	}
// 	E := g.wd.Entities[ID]
// 	if E != nil {
// 		P, ok := E.(*Player)
// 		if !ok {
// 			return nil
// 		}
// 		DeltaX, err := pk.UnpackInt16(r)
// 		if err != nil {
// 			return err
// 		}
// 		DeltaY, err := pk.UnpackInt16(r)
// 		if err != nil {
// 			return err
// 		}
// 		DeltaZ, err := pk.UnpackInt16(r)
// 		if err != nil {
// 			return err
// 		}

// 		yaw, err := r.ReadByte()
// 		if err != nil {
// 			return err
// 		}

// 		pitch, err := r.ReadByte()
// 		if err != nil {
// 			return err
// 		}
// 		P.Yaw += float32(yaw) * (1.0 / 256)
// 		P.Pitch += float32(pitch) * (1.0 / 256)

// 		og, err := r.ReadByte()
// 		if err != nil {
// 			return err
// 		}
// 		P.OnGround = og != 0x00

// 		P.X += float64(DeltaX) / 128
// 		P.Y += float64(DeltaY) / 128
// 		P.Z += float64(DeltaZ) / 128
// 	}
// 	return nil
// }

// func handleEntityHeadLookPacket(g *Client, r *bytes.Reader) {

// }

// func handleEntityRelativeMovePacket(g *Client, r *bytes.Reader) error {
// 	ID, err := pk.UnpackVarInt(r)
// 	if err != nil {
// 		return err
// 	}
// 	E := g.wd.Entities[ID]
// 	if E != nil {
// 		P, ok := E.(*Player)
// 		if !ok {
// 			return nil
// 		}
// 		DeltaX, err := pk.UnpackInt16(r)
// 		if err != nil {
// 			return err
// 		}
// 		DeltaY, err := pk.UnpackInt16(r)
// 		if err != nil {
// 			return err
// 		}
// 		DeltaZ, err := pk.UnpackInt16(r)
// 		if err != nil {
// 			return err
// 		}

// 		og, err := r.ReadByte()
// 		if err != nil {
// 			return err
// 		}
// 		P.OnGround = og != 0x00

// 		P.X += float64(DeltaX) / 128
// 		P.Y += float64(DeltaY) / 128
// 		P.Z += float64(DeltaZ) / 128
// 	}
// 	return nil
// }

func handleKeepAlivePacket(c *Client, p pk.Packet) error {
	var KeepAliveID pk.Long
	if err := p.Scan(&KeepAliveID); err != nil {
		return err
	}
	sendKeepAlivePacket(c, KeepAliveID)
	return nil
}

// func handleEntityPacket(g *Client, r *bytes.Reader) {
// 	// initialize an entity.
// }

// func handleSpawnPlayerPacket(g *Client, r *bytes.Reader) (err error) {
// 	np := new(Player)
// 	np.entityID, err = pk.UnpackVarInt(r)
// 	if err != nil {
// 		return
// 	}
// 	np.UUID[0], err = pk.UnpackInt64(r)
// 	if err != nil {
// 		return
// 	}
// 	np.UUID[1], err = pk.UnpackInt64(r)
// 	if err != nil {
// 		return
// 	}
// 	np.X, err = pk.UnpackDouble(r)
// 	if err != nil {
// 		return
// 	}
// 	np.Y, err = pk.UnpackDouble(r)
// 	if err != nil {
// 		return
// 	}
// 	np.Z, err = pk.UnpackDouble(r)
// 	if err != nil {
// 		return
// 	}

// 	yaw, err := r.ReadByte()
// 	if err != nil {
// 		return err
// 	}

// 	pitch, err := r.ReadByte()
// 	if err != nil {
// 		return err
// 	}

// 	np.Yaw = float32(yaw) * (1.0 / 256)
// 	np.Pitch = float32(pitch) * (1.0 / 256)

// 	g.wd.Entities[np.entityID] = np //把该玩家添加到全局实体表里面
// 	return nil
// }

func handleWindowItemsPacket(g *Client, p pk.Packet) (err error) {
	var (
		WindowID pk.Byte
		solts    Solts
	)
	err = p.Scan(&WindowID, &solts)
	if err != nil {
		return
	}

	switch WindowID {
	case 0: //is player inventory
		g.player.Inventory = solts
	}
	return nil
}

// func sendTeleportConfirmPacket(g *Client, TeleportID int32) {
// 	g.sendChan <- pk.Packet{
// 		ID:   0x00,
// 		Data: pk.PackVarInt(TeleportID),
// 	}
// }

func sendPlayerPositionAndLookPacket(c *Client) {
	var data []byte
	data = append(data, pk.PackDouble(g.player.X)...)
	data = append(data, pk.PackDouble(g.player.Y)...)
	data = append(data, pk.PackDouble(g.player.Z)...)
	data = append(data, pk.PackFloat(g.player.Yaw)...)
	data = append(data, pk.PackFloat(g.player.Pitch)...)
	data = append(data, pk.PackBoolean(g.player.OnGround))

	g.sendChan <- pk.Packet{
		ID:   0x11,
		Data: data,
	}

	c.conn.WritePacket(pk.Marshal(
		data.PlayerPositionAndLookServerbound,
		pk.Double(c.player.X),
		pk.Double(c.player.Y),
		pk.Double(c.player.Z),
		pk.Float(c.player.Yaw),
		pk.Float(c.player.Pitch),
		pk.Boolean(c.player.OnGround),
	))
}

// func sendPlayerLookPacket(g *Client) {
// 	var data []byte
// 	data = append(data, pk.PackFloat(g.player.Yaw)...)
// 	data = append(data, pk.PackFloat(g.player.Pitch)...)
// 	data = append(data, pk.PackBoolean(g.player.OnGround))
// 	g.sendChan <- pk.Packet{
// 		ID:   0x12,
// 		Data: data,
// 	}
// }

// func sendPlayerPositionPacket(g *Client) {
// 	var data []byte
// 	data = append(data, pk.PackDouble(g.player.X)...)
// 	data = append(data, pk.PackDouble(g.player.Y)...)
// 	data = append(data, pk.PackDouble(g.player.Z)...)
// 	data = append(data, pk.PackBoolean(g.player.OnGround))

// 	g.sendChan <- pk.Packet{
// 		ID:   0x10,
// 		Data: data,
// 	}
// }

func sendKeepAlivePacket(g *Client, KeepAliveID int64) {
	g.sendChan <- pk.Packet{
		ID:   0x0E,
		Data: pk.PackUint64(uint64(KeepAliveID)),
	}
}

// func sendClientStatusPacket(g *Client, status int32) {
// 	data := pk.PackVarInt(status)
// 	g.sendChan <- pk.Packet{
// 		ID:   0x03,
// 		Data: data,
// 	}
// }

// //hand could be 0: main hand, 1: off hand
// func sendAnimationPacket(g *Client, hand int32) {
// 	data := pk.PackVarInt(hand)
// 	g.sendChan <- pk.Packet{
// 		ID:   0x27,
// 		Data: data,
// 	}
// }

// func sendPlayerDiggingPacket(g *Client, status int32, x, y, z int, face Face) {
// 	data := pk.PackVarInt(status)
// 	data = append(data, pk.PackPosition(x, y, z)...)
// 	data = append(data, byte(face))

// 	g.sendChan <- pk.Packet{
// 		ID:   0x18,
// 		Data: data,
// 	}
// }

// func sendUseItemPacket(g *Client, hand int32) {
// 	data := pk.PackVarInt(hand)
// 	g.sendChan <- pk.Packet{
// 		ID:   0x2A,
// 		Data: data,
// 	}
// }
