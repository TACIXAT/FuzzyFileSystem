package main

import (
	"errors"
	"math/rand"
)

// https://lcamtuf.blogspot.com/2014/08/binary-fuzzing-strategies-what-works.html
type Mutator interface {
	Generate(data []byte, mapping map[int]int, name string) error
	Synthesize(data []byte, name string) []byte
	Remove(name string)
}

func NewStrats() map[string]Mutator {
	strats := make(map[string]Mutator)
	strats["bit_flip"] = NewBitFlip()
	return strats
}

type ByteDelta struct {
	Offset uint64
	Value  uint8
}

type RandByte struct {
	Diffs map[string][]ByteDelta
}

func (rb *RandByte) Generate(data []byte, mapping map[int]int, name string) error {
	if len(data) < 1 {
		return errors.New("zero size data")
	}

	rb.Diffs[name] = make([]ByteDelta, 0)

	count := 1 + int(rand.Uint64()%8)
	for i := 0; i < count; i++ {
		rb.Diffs[name] = append(rb.Diffs[name], ByteDelta{
			Offset: rand.Uint64() % uint64(len(data)),
			Value:  uint8(rand.Uint32() & 0xFF),
		})
	}

	return nil
}

func (rb *RandByte) Synthesize(data []byte, name string) []byte {
	deltas, ok := rb.Diffs[name]
	if !ok {
		return data
	}

	sz := len(data)
	bs := make([]byte, sz, sz)
	copy(bs, data)

	for _, bd := range deltas {
		bs[bd.Offset] = bd.Value
	}

	return bs
}

func (rb *RandByte) Remove(name string) {
	delete(rb.Diffs, name)
}

type BitFlip struct {
	Flips map[string]uint64
}

func NewBitFlip() *BitFlip {
	return &BitFlip{
		Flips: make(map[string]uint64),
	}
}

func (bf *BitFlip) Generate(data []byte, mapping map[int]int, name string) error {
	bitc := uint64(len(data) * 8)
	if len(mapping) > 0 {
		bitc = uint64(len(mapping) * 8)
	}

	if bitc == 0 {
		return errors.New("bitcount zero")
	}

	flip := rand.Uint64() % bitc
	if len(mapping) > 0 {
		byte := flip / 8
		bit := flip % 8
		byte = uint64(mapping[int(byte)])
		flip = byte*8 + bit
	}

	bf.Flips[name] = flip

	return nil
}

func (bf *BitFlip) Synthesize(data []byte, name string) []byte {
	bitoff, ok := bf.Flips[name]
	if !ok {
		return data
	}

	// Flip a bit!
	off := bitoff / 8
	bit := bitoff % 8
	sz := len(data)
	bs := make([]byte, sz, sz)
	copy(bs, data)
	bs[off] = data[off] ^ (1 << bit)

	return bs
}

func (bf *BitFlip) Remove(name string) {
	delete(bf.Flips, name)
}
