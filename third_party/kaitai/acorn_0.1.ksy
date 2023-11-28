meta:
  id: acorn
  file-extension: acorn
  endian: le
seq:
  - id: format_version
    contents: [ 0x01, 0x0, 0x00, 0x00 ]
  - id: compiler_version
    type: u4
  - id: debug
    type: u1
  - id: num_funcs
    type: u4
  - id: funcs
    type: func
    repeat: expr
    repeat-expr: num_funcs
  - id: num_strings
    type: u4
  - id: strings
    type: strl
    repeat: expr
    repeat-expr: num_strings
  - id: num_consts
    type: u4
  - id: consts
    type: const
    repeat: expr
    repeat-expr: num_consts
  - id: num_exports
    type: u4
  - id: exports
    type: export
    repeat: expr
    repeat-expr: num_exports
enums:
  op_kind:
    1: load_local
    2: load_global
    3: load_const
    4: swap_pop
    5: apply
    6: call
    7: match
    8: jump
    9: make_object
    10: make_pattern
    11: access
    12: update
  const_kind:
    1: unit
    2: char
    3: int
    4: float
    5: string
  stack_kind:
    1: object
    2: pattern
  object_kind:
    1: list
    2: tuple
    3: record
    4: data
  pattern_kind:
    1: alias
    2: any
    3: cons
    4: const
    5: data_option
    6: list
    7: named
    8: record
    9: tuple
  packed_const_kind:
    1: int
    2: float
  swap_pop_mode:
    1: both
    2: pop
types:
  strl:
    seq:
      - id: len
        type: u4
      - id: value
        type: str
        encoding: UTF-8
        size: len
  loc:
    seq:
      - id: line
        type: u4
      - id: col
        type: u4
  const:
    seq:
      - id: kind
        type: u1
        enum: packed_const_kind
      - id: int_value
        type: s8
        if: kind == packed_const_kind::int
      - id: float_value
        type: f8
        if: kind == packed_const_kind::float
  op:
    seq:
      - id: kind
        type: u1
        enum: op_kind

      - id: b_stack_kind
        type: u1
        enum: stack_kind
        if: kind == op_kind::load_const
      - id: b_num_args
        type: u1
        if: kind == op_kind::apply or kind == op_kind::call
      - id: b_object_kind
        type: u1
        enum: object_kind
        if: kind == op_kind::make_object
      - id: b_pattern_kind
        type: u1
        enum: pattern_kind
        if: kind == op_kind::make_pattern
      - id: b_swap_pop_mode
        type: u1
        enum: swap_pop_mode
        if: kind == op_kind::swap_pop
      - contents: [ 0x00 ]
        if: kind == op_kind::load_local or kind == op_kind::load_global or kind == op_kind::match or kind == op_kind::jump or kind == op_kind::access or kind == op_kind::update

      - id: c_const_kind
        type: u1
        enum: const_kind
        if: kind == op_kind::load_const
      - id: c_num_nested
        type: u1
        if: kind == op_kind::make_pattern
      - contents: [ 0x00 ]
        if: kind != op_kind::load_const and kind != op_kind::make_pattern

      - contents: [ 0x00 ]

      - id: a_string_hash
        type: u4
        if: kind == op_kind::load_local or kind == op_kind::call or kind == op_kind::make_pattern or kind == op_kind::access or kind == op_kind::update
      - id: a_pointer
        type: u4
        if: kind == op_kind::load_global
      - id: a_jump_delta
        type: u4
        if: kind == op_kind::jump or kind == op_kind::match
      - id: a_num_items
        type: u4
        if: kind == op_kind::make_object
      - id: a_const_pointer_value_hash
        type: u4
        if: kind == op_kind::load_const
      - contents: [ 0x00, 0x00, 0x00, 0x00 ]
        if: kind == op_kind::apply or kind == op_kind::swap_pop
  func:
    seq:
      - id: num_args
        type: u4
      - id: num_ops
        type: u4
      - id: ops
        type: op
        repeat: expr
        repeat-expr: num_ops
      - id: file_path
        type: strl
        if: _root.debug != 0
      - id: locations
        type: loc
        repeat: expr
        repeat-expr: num_ops
        if: _root.debug != 0
  export:
    seq:
      - id: name
        type: strl
      - id: address
        type: u4
