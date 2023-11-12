package bytecode

import (
	"encoding/binary"
	"io"
	"oak-compiler/ast"
	"oak-compiler/common"
	"slices"
)

type Binary struct {
	Funcs   []Func
	Strings []string
	Consts  []PackedConst
	Exports map[ast.ExternalIdentifier]Pointer

	FuncsMap  map[ast.PathIdentifier]Pointer
	StringMap map[string]StringHash
	ConstMap  map[PackedConst]ConstHash

	CompiledPaths []string
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
	NumArgs   uint32
	Ops       []Op
	FilePath  string
	Locations []ast.Location
}

func (b *Binary) Build(writer io.Writer, debug bool) {
	order := binary.LittleEndian
	w := func(v any) {
		if err := binary.Write(writer, order, v); err != nil {
			panic(common.SystemError{Message: err.Error()})
		}
	}
	ws := func(v string) {
		bs := []byte(v)
		w(uint32(len(bs)))
		w(bs)
	}
	w(uint32(common.BinaryFormatVersion))
	w(uint32(common.CompilerVersion))
	w(bool(debug))

	w(uint32(len(b.Funcs)))
	for _, fn := range b.Funcs {
		w(uint32(fn.NumArgs))
		w(uint32(len(fn.Ops)))
		for _, op := range fn.Ops {
			w(uint64(op.Word()))
		}
		if debug {
			ws(fn.FilePath)
			for _, loc := range fn.Locations {
				l, c := loc.GetLineAndColumn()
				w(uint32(l))
				w(uint32(c))
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

	var names []ast.ExternalIdentifier
	for n := range b.Exports {
		names = append(names, n)
	}
	slices.Sort(names)

	w(uint32(len(b.Exports)))
	for _, n := range names {
		ws(string(n))
		w(uint32(b.Exports[n]))
	}
}
