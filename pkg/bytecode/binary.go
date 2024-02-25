package bytecode

import (
	"encoding/binary"
	"fmt"
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

func NewBinary() *Binary {
	return &Binary{
		Exports: map[FullIdentifier]Pointer{},
	}
}

type Binary struct {
	Funcs    []Func
	Strings  []string
	Consts   []PackedConst
	Exports  map[FullIdentifier]Pointer
	Entry    FullIdentifier
	Packages []QualifiedIdentifier
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
	ws(string(b.Entry))

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

	ws(string(b.Entry))
	slices.Sort(b.Packages)
	w(uint32(len(b.Packages)))
	for _, p := range b.Packages {
		ws(string(p))
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
	if formatVersion != BinaryFormatVersion {
		return nil, fmt.Errorf("unsupported binary format version: %d", formatVersion)
	}
	var compilerVersion uint32
	e(binary.Read(reader, order, &compilerVersion))
	var debug bool
	e(binary.Read(reader, order, &debug))

	entry, err := rs(reader, order)
	if err != nil {
		return nil, err
	}
	bin.Entry = FullIdentifier(entry)

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

	entry, err = rs(reader, order)
	e(err)
	bin.Entry = FullIdentifier(entry)

	var numPackages uint32
	e(binary.Read(reader, order, &numPackages))
	bin.Packages = make([]QualifiedIdentifier, 0, numPackages)
	for i := uint32(0); i < numPackages; i++ {
		p, err := rs(reader, order)
		e(err)
		bin.Packages = append(bin.Packages, QualifiedIdentifier(p))
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
