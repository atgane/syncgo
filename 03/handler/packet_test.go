package handler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	sizeStatus    = 2
	boundStatus   = sizeStatus
	sizeSession   = 8
	boundSession  = boundStatus + sizeSession
	sizeOffset    = 4
	boundOffset   = boundSession + sizeBodySize
	sizeBodySize  = 8
	boundBodySize = boundOffset + sizeSendAt
	sizeSendAt    = 8
	boundSendAt   = boundBodySize + sizeDummy
	sizeDummy     = 8
	boundDummy    = boundSendAt + sizeDummy

	SizeHeader  = boundDummy
	maxByteSize = 6000
)

type testPacket struct {
	Header testHeader
	Body   []byte
}

type testHeader struct {
	Status   uint16
	Session  uint64
	Offset   uint32
	BodySize uint64
	SendAt   uint64
	Dummy    uint64
}

func newTestPacket(status uint16, session uint64, offset uint32, body []byte) *testPacket {
	p := &testPacket{
		Header: testHeader{
			Status:   status,
			Session:  session,
			Offset:   offset,
			BodySize: uint64(len(body)),
			SendAt:   uint64(time.Now().Unix()),
			Dummy:    0,
		},
		Body: body,
	}
	return p
}

func marshal(p *testPacket) ([]byte, error) {
	if p.Header.BodySize != uint64(len(p.Body)) {
		return nil, fmt.Errorf("invalid body size")
	}
	if p.Header.BodySize > maxByteSize {
		return nil, fmt.Errorf("exceed max body size")
	}
	h := make([]byte, SizeHeader)
	binary.LittleEndian.PutUint16(h[:boundStatus], p.Header.Status)
	binary.LittleEndian.PutUint64(h[boundStatus:boundSession], p.Header.Session)
	binary.LittleEndian.PutUint32(h[boundSession:boundOffset], p.Header.Offset)
	binary.LittleEndian.PutUint64(h[boundOffset:boundBodySize], p.Header.BodySize)
	binary.LittleEndian.PutUint64(h[boundBodySize:boundSendAt], p.Header.SendAt)
	binary.LittleEndian.PutUint64(h[boundSendAt:boundDummy], p.Header.Dummy)

	b := make([]byte, 0, SizeHeader+len(p.Body))
	b = append(b, h...)
	b = append(b, p.Body...)

	return b, nil
}

func unmarshal(b []byte) (*testPacket, int) {
	if len(b) < SizeHeader {
		return nil, 0
	}

	p := &testPacket{}
	p.Header.Status = binary.LittleEndian.Uint16(b[:boundStatus])
	p.Header.Session = binary.LittleEndian.Uint64(b[boundStatus:boundSession])
	p.Header.Offset = binary.LittleEndian.Uint32(b[boundSession:boundOffset])
	p.Header.BodySize = binary.LittleEndian.Uint64(b[boundOffset:boundBodySize])
	p.Header.SendAt = binary.LittleEndian.Uint64(b[boundBodySize:boundSendAt])
	p.Header.Dummy = binary.LittleEndian.Uint64(b[boundSendAt:boundDummy])

	if len(b) < int(SizeHeader+p.Header.BodySize) {
		return nil, 0
	}
	p.Body = b[SizeHeader : SizeHeader+int(p.Header.BodySize)]

	return p, int(SizeHeader + p.Header.BodySize)
}

func TestMarshalUnmarshalTestPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *testPacket
		padding []byte
		wantErr bool
	}{
		{
			name: "Valid packet with small body",
			packet: &testPacket{
				Header: testHeader{
					Status:   1,
					Session:  123456789,
					Offset:   42,
					BodySize: 6,
					SendAt:   987654321,
					Dummy:    5555,
				},
				Body: []byte("abcdef"),
			},
			padding: []byte("test"),
			wantErr: false,
		},
		{
			name: "Valid packet with empty body",
			packet: &testPacket{
				Header: testHeader{
					Status:   2,
					Session:  987654321,
					Offset:   84,
					BodySize: 0,
					SendAt:   123456789,
					Dummy:    0,
				},
				Body: []byte{},
			},
			padding: []byte(""),
			wantErr: false,
		},
		{
			name: "Packet with body size mismatch",
			packet: &testPacket{
				Header: testHeader{
					Status:   3,
					Session:  111111111,
					Offset:   999,
					BodySize: 5, // Body size is 5, but actual body length is 6
					SendAt:   333333333,
					Dummy:    7777,
				},
				Body: []byte("123456"), // length is 6
			},
			padding: []byte("test-test"),
			wantErr: true,
		},
		{
			name: "Packet with oversized body",
			packet: &testPacket{
				Header: testHeader{
					Status:   4,
					Session:  222222222,
					Offset:   111,
					BodySize: maxByteSize + 1, // Exceeds max byte size
					SendAt:   444444444,
					Dummy:    8888,
				},
				Body: make([]byte, maxByteSize+1), // Create an oversized body
			},
			padding: []byte("hello-world"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 직렬화 시도
			serialized, err := marshal(tt.packet)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				return
			}

			serializedWithPadding := append(serialized, tt.padding...)

			deserializedPacket, packetLen := unmarshal(serializedWithPadding)
			require.Equal(t, len(serialized), packetLen)
			require.Equal(t, tt.packet.Header, deserializedPacket.Header)
			require.True(t, bytes.Equal(tt.packet.Body, deserializedPacket.Body))
		})
	}
}
