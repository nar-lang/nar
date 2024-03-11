package bytecode

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"slices"
)

func Version() int { return 100 }

const signature = 'N'<<8 | 'A'<<16 | 'R'<<24

type QualifiedIdentifier string

type FullIdentifier string

type Location struct {
	Line, Column uint32
}

func NewBinary() *Binary {
	return &Binary{
		Exports:  map[FullIdentifier]Pointer{},
		Packages: map[QualifiedIdentifier]int32{},
	}
}

type Binary struct {
	CompilerVersion uint32
	Funcs           []Func
	Strings         []string
	Consts          []PackedConst
	Exports         map[FullIdentifier]Pointer
	Entry           FullIdentifier
	Packages        map[QualifiedIdentifier]int32
}

type Func struct {
	Name      StringHash
	NumArgs   uint32
	Ops       []Op
	FilePath  string
	Locations []Location
}

func (b *Binary) Write(writer io.Writer, debug bool) (err error) {
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
	w(uint32(signature))
	w(uint32(Version()))
	w(uint32(b.CompilerVersion))
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

	packageNames := make([]QualifiedIdentifier, 0, len(b.Packages))
	for p := range b.Packages {
		packageNames = append(packageNames, p)
	}
	slices.Sort(packageNames)
	w(uint32(len(b.Packages)))
	for _, p := range packageNames {
		ws(string(p))
		w(int32(b.Packages[p]))
	}

	return nil
}

func Read(reader io.Reader) (bin *Binary, err error) {
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
	var sign uint32
	e(binary.Read(reader, order, &sign))
	if sign != signature {
		return nil, fmt.Errorf("invalid file format")
	}
	var formatVersion uint32
	e(binary.Read(reader, order, &formatVersion))
	if int(formatVersion) != Version() {
		return nil, fmt.Errorf("unsupported binary format version: %d", formatVersion)
	}
	e(binary.Read(reader, order, &bin.CompilerVersion))
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

	var numPackages uint32
	e(binary.Read(reader, order, &numPackages))
	bin.Packages = make(map[QualifiedIdentifier]int32, numPackages)
	for i := uint32(0); i < numPackages; i++ {
		name, err := rs(reader, order)
		e(err)
		var version int32
		e(binary.Read(reader, order, &version))
		bin.Packages[QualifiedIdentifier(name)] = version
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
