package bytecode

import (
	"encoding/binary"
	"io"
	"math"
	"slices"
)

const BinaryFormatVersion uint32 = 100

type QualifiedIdentifier string

type FullIdentifier string

type Location struct {
	Line, Column uint32
}

type Binary struct {
	Funcs   []Func
	Strings []string
	Consts  []PackedConst
	Exports map[FullIdentifier]Pointer

	FuncsMap  map[FullIdentifier]Pointer
	StringMap map[string]StringHash
	ConstMap  map[PackedConst]ConstHash

	CompiledPaths []QualifiedIdentifier
}

func (b *Binary) HashString(v string) StringHash {
	if h, ok := b.StringMap[v]; ok {
		return h
	}
	hash := StringHash(len(b.StringMap))
	b.StringMap[v] = hash
	b.Strings = append(b.Strings, v)
	return hash
}

func (b *Binary) HashConst(v PackedConst) ConstHash {
	if h, ok := b.ConstMap[v]; ok {
		return h
	}
	hash := ConstHash(len(b.ConstMap))
	b.ConstMap[v] = hash
	b.Consts = append(b.Consts, v)
	return hash
}

type Func struct {
	Name      StringHash
	NumArgs   uint32
	Ops       []Op
	FilePath  string
	Locations []Location
}

func (b *Binary) Build(writer io.Writer, compilerVersion uint32, debug bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	order := binary.LittleEndian
	w := func(v any) {
		if err := binary.Write(writer, order, v); err != nil {
			panic(err)
		}
	}
	ws := func(v string) {
		bs := []byte(v)
		w(uint32(len(bs)))
		w(bs)
	}
	w(uint32(BinaryFormatVersion))
	w(uint32(compilerVersion))
	w(bool(debug))

	w(uint32(len(b.Funcs)))
	for _, fn := range b.Funcs {
		w(uint32(fn.NumArgs))
		w(uint32(len(fn.Ops)))
		for _, op := range fn.Ops {
			w(uint64(op))
		}
		if debug {
			ws(fn.FilePath)
			for _, loc := range fn.Locations {
				w(uint32(loc.Line))
				w(uint32(loc.Column))
			}
		}
	}

	w(uint32(len(b.Strings)))
	for _, str := range b.Strings {
		ws(str)
	}

	w(uint32(len(b.Consts)))
	for _, c := range b.Consts {
		w(uint8(c.Kind()))
		w(uint64(c.Pack()))
	}

	var names []FullIdentifier
	for n := range b.Exports {
		names = append(names, n)
	}
	slices.Sort(names)

	w(uint32(len(b.Exports)))
	for _, n := range names {
		ws(string(n))
		w(uint32(b.Exports[n]))
	}
	return nil
}

func Load(reader io.Reader) (bin *Binary, err error) {
	bin = &Binary{}
	e := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	order := binary.LittleEndian
	rs := func(reader io.Reader, order binary.ByteOrder) (string, error) {
		var l uint32
		e(binary.Read(reader, order, &l))
		bs := make([]byte, l)
		e(binary.Read(reader, order, bs))
		return string(bs), nil
	}
	var formatVersion uint32
	e(binary.Read(reader, order, &formatVersion))
	var compilerVersion uint32
	e(binary.Read(reader, order, &compilerVersion))
	var debug bool
	e(binary.Read(reader, order, &debug))

	var numFuncs uint32
	e(binary.Read(reader, order, &numFuncs))
	bin.Funcs = make([]Func, 0, numFuncs)
	for i := uint32(0); i < numFuncs; i++ {
		var numArgs uint32
		e(binary.Read(reader, order, &numArgs))
		var numOps uint32
		e(binary.Read(reader, order, &numOps))
		ops := make([]Op, 0, numOps)
		for j := uint32(0); j < numOps; j++ {
			var word uint64
			e(binary.Read(reader, order, &word))
			ops = append(ops, Op(word))
		}
		var filePath string
		var locations []Location
		if debug {
			filePath, err = rs(reader, order)
			e(err)
			for j := uint32(0); j < numOps; j++ {
				loc := Location{}
				e(binary.Read(reader, order, &loc.Line))
				e(binary.Read(reader, order, &loc.Column))
				locations = append(locations, loc)
			}
		}
		bin.Funcs = append(bin.Funcs, Func{
			NumArgs:   numArgs,
			Ops:       ops,
			FilePath:  filePath,
			Locations: locations,
		})

	}

	var numStrings uint32
	e(binary.Read(reader, order, &numStrings))
	bin.Strings = make([]string, 0, numStrings)
	for i := uint32(0); i < numStrings; i++ {
		str, err := rs(reader, order)
		e(err)
		bin.Strings = append(bin.Strings, str)
	}

	var numConsts uint32
	e(binary.Read(reader, order, &numConsts))
	bin.Consts = make([]PackedConst, 0, numConsts)
	for i := uint32(0); i < numConsts; i++ {
		var kind uint8
		e(binary.Read(reader, order, &kind))
		var packed uint64
		e(binary.Read(reader, order, &packed))
		bin.Consts = append(bin.Consts, unpackConst(kind, packed))
	}

	var numExports uint32
	e(binary.Read(reader, order, &numExports))
	bin.Exports = make(map[FullIdentifier]Pointer, numExports)
	for i := uint32(0); i < numExports; i++ {
		name, err := rs(reader, order)
		e(err)
		var ptr uint32
		e(binary.Read(reader, order, &ptr))
		bin.Exports[FullIdentifier(name)] = Pointer(ptr)
	}

	return
}

func unpackConst(kind uint8, packed uint64) PackedConst {
	switch ConstHashKind(kind) {
	case ConstHashKindInt:
		return PackedInt{Value: int64(packed)}
	case ConstHashKindFloat:
		return PackedFloat{Value: math.Float64frombits(packed)}
	default:
		panic("unknown const kind")
	}
}
