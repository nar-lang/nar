package bytecode

type BinaryHash struct {
	FuncsMap  map[FullIdentifier]Pointer
	StringMap map[string]StringHash
	ConstMap  map[PackedConst]ConstHash

	CompiledPaths []QualifiedIdentifier
}

func NewBinaryHash() *BinaryHash {
	return &BinaryHash{
		FuncsMap:  map[FullIdentifier]Pointer{},
		StringMap: map[string]StringHash{},
		ConstMap:  map[PackedConst]ConstHash{},
	}
}

func (h *BinaryHash) HashString(v string, bin *Binary) StringHash {
	if h, ok := h.StringMap[v]; ok {
		return h
	}
	hash := StringHash(len(h.StringMap))
	h.StringMap[v] = hash
	bin.Strings = append(bin.Strings, v)
	return hash
}

func (h *BinaryHash) HashConst(v PackedConst, bin *Binary) ConstHash {
	if h, ok := h.ConstMap[v]; ok {
		return h
	}
	hash := ConstHash(len(h.ConstMap))
	h.ConstMap[v] = hash
	bin.Consts = append(bin.Consts, v)
	return hash
}
