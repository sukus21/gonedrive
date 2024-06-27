package quickxor

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

const bitsInLastCell = 32
const shift = 11
const widthInBits = 160

type Hasher struct {
	data        []uint64
	lengthSoFar int64
	shiftSoFar  int
}

func NewHasher() *Hasher {
	return &Hasher{
		data:        make([]uint64, (widthInBits-1)/64+1),
		shiftSoFar:  0,
		lengthSoFar: 0,
	}
}

// Never returns errors, always returns full length
func (qxor *Hasher) Write(buf []byte) (int, error) {
	qxor.hashCore(buf)
	return len(buf), nil
}

func (qxor *Hasher) hashCore(array []byte) {
	currentShift := qxor.shiftSoFar
	vectorArrayIndex := currentShift / 64
	vectorOffset := currentShift % 64
	iterations := min(len(array), widthInBits)

	for i := 0; i < iterations; i++ {
		isLastCell := vectorArrayIndex == len(qxor.data)-1
		bitsInVectorCell := 64
		if isLastCell {
			bitsInVectorCell = bitsInLastCell
		}

		if vectorOffset <= bitsInVectorCell-8 {
			for j := i; j < len(array); j += widthInBits {
				qxor.data[vectorArrayIndex] ^= uint64(array[j]) << vectorOffset
			}
		} else {
			low := byte(bitsInVectorCell - vectorOffset)
			index1 := vectorArrayIndex
			index2 := vectorArrayIndex + 1
			if isLastCell {
				index2 = 0
			}

			xoredByte := uint8(0)
			for j := i; j < len(array); j += widthInBits {
				xoredByte ^= array[j]
			}
			qxor.data[index1] ^= uint64(xoredByte) << vectorOffset
			qxor.data[index2] ^= uint64(xoredByte) >> low
		}

		vectorOffset += shift
		for vectorOffset >= bitsInVectorCell {
			vectorArrayIndex++
			if isLastCell {
				vectorArrayIndex = 0
			}
			vectorOffset -= bitsInVectorCell
		}
	}

	qxor.shiftSoFar = (qxor.shiftSoFar + shift*(len(array)%widthInBits)) % widthInBits
	qxor.lengthSoFar += int64(len(array))
}

func (qxor *Hasher) hashFinal() []byte {
	rgb := make([]byte, (widthInBits-1)/8+1)

	// Copy data to output
	buf := [8]byte{}
	for i := 0; i < len(qxor.data)-1; i++ {
		binary.LittleEndian.PutUint64(buf[:], qxor.data[i])
		copy(rgb[i*8:], buf[:])
	}

	// Copy trailing data to output
	binary.LittleEndian.PutUint64(buf[:], qxor.data[len(qxor.data)-1])
	length := (len(qxor.data) - 1) * 8
	copy(rgb[length:], buf[:len(rgb)-length])

	// XOR with the length of the data
	binary.LittleEndian.PutUint64(buf[:], uint64(qxor.lengthSoFar))
	for i, v := range buf {
		rgb[(widthInBits/8)-8+i] ^= v
	}

	// Return final hash
	return rgb
}

func (qxor *Hasher) GetHash() []byte {
	return qxor.hashFinal()
}

func (qxor *Hasher) GetHashBase64() string {
	hash := qxor.hashFinal()

	buf := bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, &buf)
	enc.Write(hash)
	enc.Close()
	return buf.String()
}

func QuickXorHash(array []byte) []byte {
	qxor := NewHasher()
	qxor.Write(array)
	return qxor.GetHash()
}

func QuickXorHashBase64(array []byte) string {
	qxor := NewHasher()
	qxor.Write(array)
	return qxor.GetHashBase64()
}
