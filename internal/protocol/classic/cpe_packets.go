package classic

import (
	"encoding/binary"
	"math"
)

func encodeExtInfo(appName string, count int) []byte {
	packet := make([]byte, 67)
	packet[0] = opcodeExtInfo
	writeFixedString(packet[1:65], appName)
	binary.BigEndian.PutUint16(packet[65:67], uint16(count))
	return packet
}

func encodeExtEntry(name string, version uint32) []byte {
	packet := make([]byte, 69)
	packet[0] = opcodeExtEntry
	writeFixedString(packet[1:65], name)
	binary.BigEndian.PutUint32(packet[65:69], version)
	return packet
}

func encodeExtAddPlayerName(id byte, name string) []byte {
	packet := make([]byte, 196)
	packet[0] = opcodeExtAddPlayerName
	packet[1] = id
	writeFixedString(packet[2:66], name)
	writeFixedString(packet[66:130], name)
	writeFixedString(packet[130:194], "Players")
	packet[194] = 0
	return packet
}

func encodeExtRemovePlayerName(id byte) []byte {
	packet := make([]byte, 3)
	packet[0] = opcodeExtRemovePlayerName
	packet[1] = id
	return packet
}

func encodeExtAddEntity(id byte, skin, name string) []byte {
	packet := make([]byte, 130)
	packet[0] = opcodeExtAddEntity
	packet[1] = id
	writeFixedString(packet[2:66], skin)
	writeFixedString(packet[66:130], name)
	return packet
}

func encodeTwoWayPing(serverToClient bool, id uint16) []byte {
	packet := make([]byte, 4)
	packet[0] = opcodeTwoWayPing
	if serverToClient {
		packet[1] = 1
	}
	binary.BigEndian.PutUint16(packet[2:4], id)
	return packet
}

func encodeClickDistance(distance int16) []byte {
	packet := make([]byte, 3)
	packet[0] = opcodeSetClickDistance
	binary.BigEndian.PutUint16(packet[1:3], uint16(distance))
	return packet
}

func encodeCustomBlockSupportLevel(level byte) []byte {
	return []byte{opcodeCustomBlockSupport, level}
}

func encodeHoldThis(blockID byte, locked bool) []byte {
	packet := make([]byte, 3)
	packet[0] = opcodeHoldThis
	packet[1] = blockID
	if locked {
		packet[2] = 1
	}
	return packet
}

func encodeTextHotkey(label, input string, keycode int32, mods byte) []byte {
	packet := make([]byte, 134)
	packet[0] = opcodeSetTextHotkey
	writeFixedString(packet[1:65], label)
	writeFixedString(packet[65:129], input)
	binary.BigEndian.PutUint32(packet[129:133], uint32(keycode))
	packet[133] = mods
	return packet
}

func encodeEnvColor(colorType byte, r, g, b int16) []byte {
	packet := make([]byte, 8)
	packet[0] = opcodeEnvColors
	packet[1] = colorType
	binary.BigEndian.PutUint16(packet[2:4], uint16(r))
	binary.BigEndian.PutUint16(packet[4:6], uint16(g))
	binary.BigEndian.PutUint16(packet[6:8], uint16(b))
	return packet
}

func encodeMakeSelection(id byte, label string, x1, y1, z1, x2, y2, z2 uint16, r, g, b, opacity int16) []byte {
	packet := make([]byte, 86)
	packet[0] = opcodeMakeSelection
	packet[1] = id
	writeFixedString(packet[2:66], label)
	binary.BigEndian.PutUint16(packet[66:68], x1)
	binary.BigEndian.PutUint16(packet[68:70], y1)
	binary.BigEndian.PutUint16(packet[70:72], z1)
	binary.BigEndian.PutUint16(packet[72:74], x2)
	binary.BigEndian.PutUint16(packet[74:76], y2)
	binary.BigEndian.PutUint16(packet[76:78], z2)
	binary.BigEndian.PutUint16(packet[78:80], uint16(r))
	binary.BigEndian.PutUint16(packet[80:82], uint16(g))
	binary.BigEndian.PutUint16(packet[82:84], uint16(b))
	binary.BigEndian.PutUint16(packet[84:86], uint16(opacity))
	return packet
}

func encodeRemoveSelection(id byte) []byte {
	return []byte{opcodeRemoveSelection, id}
}

func encodeSetBlockPermission(blockID byte, place, delete bool) []byte {
	packet := make([]byte, 4)
	packet[0] = opcodeSetBlockPermission
	packet[1] = blockID
	if place {
		packet[2] = 1
	}
	if delete {
		packet[3] = 1
	}
	return packet
}

func encodeChangeModel(entityID byte, model string) []byte {
	packet := make([]byte, 66)
	packet[0] = opcodeChangeModel
	packet[1] = entityID
	writeFixedString(packet[2:66], model)
	return packet
}

func encodeMapAppearance(url string, sideBlock, edgeBlock byte, edgeLevel int16) []byte {
	packet := make([]byte, 69)
	packet[0] = opcodeEnvSetMapAppearance
	writeFixedString(packet[1:65], url)
	packet[65] = sideBlock
	packet[66] = edgeBlock
	binary.BigEndian.PutUint16(packet[67:69], uint16(edgeLevel))
	return packet
}

func encodeMapAppearanceV2(url string, sideBlock, edgeBlock byte, edgeLevel, cloudsHeight, maxFog int16) []byte {
	packet := make([]byte, 73)
	packet[0] = opcodeEnvSetMapAppearance
	writeFixedString(packet[1:65], url)
	packet[65] = sideBlock
	packet[66] = edgeBlock
	binary.BigEndian.PutUint16(packet[67:69], uint16(edgeLevel))
	binary.BigEndian.PutUint16(packet[69:71], uint16(cloudsHeight))
	binary.BigEndian.PutUint16(packet[71:73], uint16(maxFog))
	return packet
}

func encodeEnvWeatherType(weatherType byte) []byte {
	return []byte{opcodeEnvWeatherType, weatherType}
}

func encodeHackControl(canFly, canNoclip, canSpeed, canRespawn, canThirdPerson bool, maxJumpHeight int16) []byte {
	packet := make([]byte, 8)
	packet[0] = opcodeHackControl
	if canFly {
		packet[1] = 1
	}
	if canNoclip {
		packet[2] = 1
	}
	if canSpeed {
		packet[3] = 1
	}
	if canRespawn {
		packet[4] = 1
	}
	if canThirdPerson {
		packet[5] = 1
	}
	binary.BigEndian.PutUint16(packet[6:8], uint16(maxJumpHeight))
	return packet
}

func encodeExtAddEntity2(entityID byte, displayName, skinName string, x, y, z int, yaw, pitch byte) []byte {
	packet := make([]byte, 138)
	packet[0] = opcodeExtAddEntity2
	packet[1] = entityID
	writeFixedString(packet[2:66], displayName)
	writeFixedString(packet[66:130], skinName)
	binary.BigEndian.PutUint16(packet[130:132], uint16(x*coordScale))
	binary.BigEndian.PutUint16(packet[132:134], uint16(y*coordScale+eyeHeight))
	binary.BigEndian.PutUint16(packet[134:136], uint16(z*coordScale))
	packet[136] = yaw
	packet[137] = pitch
	return packet
}

func encodeSetTextColor(r, g, b, a, index byte) []byte {
	return []byte{opcodeSetTextColor, r, g, b, a, index}
}

func encodeSetMapEnvURL(url string) []byte {
	packet := make([]byte, 65)
	packet[0] = opcodeSetMapEnvURL
	writeFixedString(packet[1:65], url)
	return packet
}

func encodeSetMapEnvURLV2(url string) []byte {
	packet := make([]byte, 129)
	packet[0] = opcodeSetMapEnvURL
	writeFixedString(packet[1:65], url)
	if len(url) > 64 {
		writeFixedString(packet[65:129], url[64:])
	}
	return packet
}

func encodeSetMapEnvProperty(prop byte, value int32) []byte {
	packet := make([]byte, 6)
	packet[0] = opcodeSetMapEnvProperty
	packet[1] = prop
	binary.BigEndian.PutUint32(packet[2:6], uint32(value))
	return packet
}

func encodeSetEntityProperty(entityID, prop byte, value int32) []byte {
	packet := make([]byte, 7)
	packet[0] = opcodeSetEntityProperty
	packet[1] = entityID
	packet[2] = prop
	binary.BigEndian.PutUint32(packet[3:7], uint32(value))
	return packet
}

func encodeSetInventoryOrder(blockID, order byte) []byte {
	return []byte{opcodeSetInventoryOrder, blockID, order}
}

func encodeSetHotbar(blockID, slot byte) []byte {
	return []byte{opcodeSetHotbar, blockID, slot}
}

func encodeSetSpawnpoint(x, y, z int, yaw, pitch byte) []byte {
	packet := make([]byte, 9)
	packet[0] = opcodeSetSpawnpoint
	binary.BigEndian.PutUint16(packet[1:3], uint16(x*coordScale))
	binary.BigEndian.PutUint16(packet[3:5], uint16(y*coordScale))
	binary.BigEndian.PutUint16(packet[5:7], uint16(z*coordScale))
	packet[7] = yaw
	packet[8] = pitch
	return packet
}

func encodeVelocityControl(x, y, z float32, xMode, yMode, zMode byte) []byte {
	packet := make([]byte, 16)
	packet[0] = opcodeVelocityControl
	binary.BigEndian.PutUint32(packet[1:5], uint32(int32(x*10000)))
	binary.BigEndian.PutUint32(packet[5:9], uint32(int32(y*10000)))
	binary.BigEndian.PutUint32(packet[9:13], uint32(int32(z*10000)))
	packet[13] = xMode
	packet[14] = yMode
	packet[15] = zMode
	return packet
}

func encodeDefineEffect(
	effectID, u1, v1, u2, v2, tintR, tintG, tintB,
	frameCount, particleCount, size byte,
	sizeVariation, spread, speed, gravity, baseLifetime, lifetimeVariation float32,
	expireUponTouchingGround, collidesSolid, collidesLiquid, collidesLeaves, fullBright bool,
) []byte {
	packet := make([]byte, 36)
	packet[0] = opcodeDefineEffect
	packet[1] = effectID
	packet[2] = u1
	packet[3] = v1
	packet[4] = u2
	packet[5] = v2
	packet[6] = tintR
	packet[7] = tintG
	packet[8] = tintB
	packet[9] = frameCount
	packet[10] = particleCount
	packet[11] = size
	binary.BigEndian.PutUint32(packet[12:16], uint32(int32(sizeVariation*10000)))
	binary.BigEndian.PutUint16(packet[16:18], uint16(spread*32))
	binary.BigEndian.PutUint32(packet[18:22], uint32(int32(speed*10000)))
	binary.BigEndian.PutUint32(packet[22:26], uint32(int32(gravity*10000)))
	binary.BigEndian.PutUint32(packet[26:30], uint32(int32(baseLifetime*10000)))
	binary.BigEndian.PutUint32(packet[30:34], uint32(int32(lifetimeVariation*10000)))
	var flags byte
	if expireUponTouchingGround {
		flags |= 1 << 0
	}
	if collidesSolid {
		flags |= 1 << 1
	}
	if collidesLiquid {
		flags |= 1 << 2
	}
	if collidesLeaves {
		flags |= 1 << 3
	}
	packet[34] = flags
	if fullBright {
		packet[35] = 1
	}
	return packet
}

func encodeSpawnEffect(effectID byte, x, y, z, originX, originY, originZ float32) []byte {
	packet := make([]byte, 26)
	packet[0] = opcodeSpawnEffect
	packet[1] = effectID
	binary.BigEndian.PutUint32(packet[2:6], uint32(int32(x*32)))
	binary.BigEndian.PutUint32(packet[6:10], uint32(int32(y*32)))
	binary.BigEndian.PutUint32(packet[10:14], uint32(int32(z*32)))
	binary.BigEndian.PutUint32(packet[14:18], uint32(int32(originX*32)))
	binary.BigEndian.PutUint32(packet[18:22], uint32(int32(originY*32)))
	binary.BigEndian.PutUint32(packet[22:26], uint32(int32(originZ*32)))
	return packet
}

func encodeDefineModel(modelID byte, name string, bobbing, pushes, usesHumanSkin, calcHumanAnims bool, nameY, eyeY float32, collisionX, collisionY, collisionZ float32, pickMinX, pickMinY, pickMinZ, pickMaxX, pickMaxY, pickMaxZ float32, uScale, vScale uint16, partCount byte) []byte {
	packet := make([]byte, 116)
	packet[0] = opcodeDefineModel
	packet[1] = modelID
	writeFixedString(packet[2:66], name)
	var flags byte
	if bobbing {
		flags |= 1 << 0
	}
	if pushes {
		flags |= 1 << 1
	}
	if usesHumanSkin {
		flags |= 1 << 2
	}
	if calcHumanAnims {
		flags |= 1 << 3
	}
	packet[66] = flags
	binary.BigEndian.PutUint32(packet[67:71], math.Float32bits(nameY))
	binary.BigEndian.PutUint32(packet[71:75], math.Float32bits(eyeY))
	binary.BigEndian.PutUint32(packet[75:79], math.Float32bits(collisionX))
	binary.BigEndian.PutUint32(packet[79:83], math.Float32bits(collisionY))
	binary.BigEndian.PutUint32(packet[83:87], math.Float32bits(collisionZ))
	binary.BigEndian.PutUint32(packet[87:91], math.Float32bits(pickMinX))
	binary.BigEndian.PutUint32(packet[91:95], math.Float32bits(pickMinY))
	binary.BigEndian.PutUint32(packet[95:99], math.Float32bits(pickMinZ))
	binary.BigEndian.PutUint32(packet[99:103], math.Float32bits(pickMaxX))
	binary.BigEndian.PutUint32(packet[103:107], math.Float32bits(pickMaxY))
	binary.BigEndian.PutUint32(packet[107:111], math.Float32bits(pickMaxZ))
	binary.BigEndian.PutUint16(packet[111:113], uScale)
	binary.BigEndian.PutUint16(packet[113:115], vScale)
	packet[115] = partCount
	return packet
}

func encodeUndefineModel(modelID byte) []byte {
	return []byte{opcodeUndefineModel, modelID}
}

func encodePluginMessage(channel byte, data []byte) []byte {
	packet := make([]byte, 66)
	packet[0] = opcodePluginMessage
	packet[1] = channel
	copy(packet[2:66], data)
	return packet
}

func encodeEntityTeleportExt(entityID byte, usePos bool, moveMode byte, useOri, interpolateOri bool, x, y, z int, yaw, pitch byte) []byte {
	packet := make([]byte, 11)
	packet[0] = opcodeEntityTeleportExt
	packet[1] = entityID
	var flags byte
	if usePos {
		flags |= 1
	}
	flags |= (moveMode & 0x0F) << 1
	if useOri {
		flags |= 16
	}
	if interpolateOri {
		flags |= 32
	}
	packet[2] = flags
	binary.BigEndian.PutUint16(packet[3:5], uint16(x*coordScale))
	binary.BigEndian.PutUint16(packet[5:7], uint16(y*coordScale+eyeHeight))
	binary.BigEndian.PutUint16(packet[7:9], uint16(z*coordScale))
	packet[9] = yaw
	packet[10] = pitch
	return packet
}

func encodeLightingMode(mode byte, locked bool) []byte {
	packet := make([]byte, 3)
	packet[0] = opcodeLightingMode
	packet[1] = mode
	if locked {
		packet[2] = 1
	}
	return packet
}

func encodeCinematicGui(hideCrosshair, hideHand, hideHotbar bool, r, g, b, opacity byte, barSize uint16) []byte {
	packet := make([]byte, 10)
	packet[0] = opcodeCinematicGui
	if hideCrosshair {
		packet[1] = 1
	}
	if hideHand {
		packet[2] = 1
	}
	if hideHotbar {
		packet[3] = 1
	}
	packet[4] = r
	packet[5] = g
	packet[6] = b
	packet[7] = opacity
	binary.BigEndian.PutUint16(packet[8:10], barSize)
	return packet
}

func encodeToggleBlockList(toggle bool) []byte {
	packet := make([]byte, 2)
	packet[0] = opcodeToggleBlockList
	if toggle {
		packet[1] = 1
	}
	return packet
}

func encodeBulkBlockUpdate(entries []bulkBlockEntry) []byte {
	if len(entries) > 256 {
		entries = entries[:256]
	}
	packet := make([]byte, 1+1+len(entries)*4+1+((len(entries)+7)/8))
	packet[0] = opcodeBulkBlockUpdate
	packet[1] = byte(len(entries))
	for i, e := range entries {
		binary.BigEndian.PutUint16(packet[2+i*4:4+i*4], uint16(e.Index))
		packet[4+i*4] = e.BlockID
	}
	lightFlags := 2 + len(entries)*4
	packet[lightFlags] = byte((len(entries) + 7) / 8)
	return packet
}

type bulkBlockEntry struct {
	Index   int
	BlockID byte
}

func encodeDefineBlock(
	blockID byte, name string, collideType, rawSpeed byte,
	topTex, sideTex, bottomTex byte,
	blocksLight bool, walkSound byte, brightness byte,
	blockDraw, fogDensity, fogR, fogG, fogB byte,
	extTex bool,
) []byte {
	texSize := 1
	if extTex {
		texSize = 2
	}
	packet := make([]byte, 80+3*(texSize-1))
	packet[0] = opcodeDefineBlock
	packet[1] = blockID
	writeFixedString(packet[2:66], name)
	packet[66] = collideType
	packet[67] = rawSpeed
	off := 68
	writeTex(packet[off:], topTex, extTex); off += texSize
	writeTex(packet[off:], sideTex, extTex); off += texSize
	writeTex(packet[off:], bottomTex, extTex); off += texSize
	if !blocksLight {
		packet[off] = 1
	}
	off++
	packet[off] = walkSound; off++
	packet[off] = brightness; off++
	packet[off] = blockDraw; off++
	packet[off] = fogDensity; off++
	packet[off] = fogR; off++
	packet[off] = fogG; off++
	packet[off] = fogB
	return packet
}

func encodeUndefineBlock(blockID byte) []byte {
	return []byte{opcodeUndefineBlock, blockID}
}

func encodeDefineBlockExt(
	blockID byte, name string, collideType, rawSpeed byte,
	topTex, leftTex, rightTex, frontTex, backTex, bottomTex byte,
	blocksLight bool, walkSound byte, brightness byte,
	minX, minZ, minY, maxX, maxZ, maxY byte,
	blockDraw, fogDensity, fogR, fogG, fogB byte,
	extTex bool,
) []byte {
	texSize := 1
	if extTex {
		texSize = 2
	}
	packet := make([]byte, 88+6*(texSize-1))
	packet[0] = opcodeDefineBlockExt
	packet[1] = blockID
	writeFixedString(packet[2:66], name)
	packet[66] = collideType
	packet[67] = rawSpeed
	off := 68
	writeTex(packet[off:], topTex, extTex); off += texSize
	writeTex(packet[off:], leftTex, extTex); off += texSize
	writeTex(packet[off:], rightTex, extTex); off += texSize
	writeTex(packet[off:], frontTex, extTex); off += texSize
	writeTex(packet[off:], backTex, extTex); off += texSize
	writeTex(packet[off:], bottomTex, extTex); off += texSize
	if !blocksLight {
		packet[off] = 1
	}
	off++
	packet[off] = walkSound; off++
	packet[off] = brightness; off++
	packet[off] = blockDraw; off++
	packet[off] = fogDensity; off++
	packet[off] = fogR; off++
	packet[off] = fogG; off++
	packet[off] = fogB; off++
	packet[off] = minX; off++
	packet[off] = minZ; off++
	packet[off] = minY; off++
	packet[off] = maxX; off++
	packet[off] = maxZ; off++
	packet[off] = maxY
	return packet
}

func writeTex(buf []byte, tex byte, extTex bool) {
	if extTex {
		buf[0] = 0
		buf[1] = tex
	} else {
		buf[0] = tex
	}
}
