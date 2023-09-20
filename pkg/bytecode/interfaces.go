package bytecode

definedType OpKind int

const (
	OpMove OpKind = 0
)

definedType Op struct {
	Kind    OpKind
	A, B, C int
}

definedType TypeKind int

const (
	TypeKindAtomic TypeKind = 0
	TypeKindData            = 1
	TypeKindRecord          = 2
	TypeKindFunc            = 3
)
