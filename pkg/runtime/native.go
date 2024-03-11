package runtime

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include "../../../narc/nar.h"

typedef int (*init_fn_t)(init_data_t*);
int init_wrapper(void* init_fn, nar_runtime_t runtime);

typedef nar_object_t (*func0)(void);
typedef nar_object_t (*func1)(nar_object_t);
typedef nar_object_t (*func2)(nar_object_t, nar_object_t);
typedef nar_object_t (*func3)(nar_object_t, nar_object_t, nar_object_t);
typedef nar_object_t (*func4)(nar_object_t, nar_object_t, nar_object_t, nar_object_t);
typedef nar_object_t (*func5)(nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t);
typedef nar_object_t (*func6)(nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t);
typedef nar_object_t (*func7)(nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t);
typedef nar_object_t (*func8)(nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t, nar_object_t);

nar_object_t call_func0(func0 fn);
nar_object_t call_func1(func1 fn, nar_object_t a);
nar_object_t call_func2(func2 fn, nar_object_t a, nar_object_t b);
nar_object_t call_func3(func3 fn, nar_object_t a, nar_object_t b, nar_object_t c);
nar_object_t call_func4(func4 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d);
nar_object_t call_func5(func5 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e);
nar_object_t call_func6(func6 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e, nar_object_t f);
nar_object_t call_func7(func7 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e, nar_object_t f, nar_object_t g);
nar_object_t call_func8(func8 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e, nar_object_t f, nar_object_t g, nar_object_t h);
*/
import "C"
import (
	"fmt"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/pkg/bytecode"
	"os"
	"path/filepath"
	"unsafe"
)

import (
	goruntime "runtime"
)

type runtimeId uint32

var runtimeIds = map[*Runtime]runtimeId{}
var runtimes []*Runtime
var funcWrappers = map[C.nar_func_ptr_t]unsafe.Pointer{}

func getRuntimeId(rt *Runtime) C.nar_runtime_t {
	id, ok := runtimeIds[rt]
	if !ok {
		id = runtimeId(len(runtimes))
		runtimeIds[rt] = id
		runtimes = append(runtimes, rt)
	}
	return C.nar_runtime_t(id)
}

func getRuntime(id C.nar_runtime_t) *Runtime {
	return runtimes[id]
}

func RegisterNativeLibrary(runtime *Runtime, packageName string, packageVersion int, libsPath string) error {
	var initPtr unsafe.Pointer
	if goruntime.GOOS == "darwin" {
		libPath := filepath.Join(libsPath, fmt.Sprintf("lib%s.%d.dylib", packageName, packageVersion))
		if _, err := os.Stat(libPath); err != nil {
			return nil
		}

		handle := C.dlopen(C.CString(libPath), C.RTLD_LAZY)
		if handle == nil {
			return fmt.Errorf("dlopen failed: %s", (string)(C.GoString(C.dlerror())))
		}
		initPtr = C.dlsym(handle, C.CString("init"))
		if initPtr == nil {
			return fmt.Errorf("dlsym failed: %s", (string)(C.GoString(C.dlerror())))
		}
	}

	result := C.init_wrapper(initPtr, getRuntimeId(runtime))
	if result != 0 {
		return fmt.Errorf("failed to init package %s with code %d", packageName, int(result))
	}
	return nil
}

func objToGo(obj C.nar_object_t) Object {
	return Object(obj)
}

func objToC(obj Object) C.nar_object_t {
	return C.nar_object_t(obj)
}

var decoder = func() *encoding.Decoder {
	if goruntime.GOOS == "windows" {
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	} else {
		return utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder()
	}
}()

var stringStride = func() int {
	if goruntime.GOOS == "windows" {
		return 2
	} else {
		return 4
	}
}()

func stringToGo(s C.nar_string_t) TString {
	strSlice := (*[1 << 30]C.char)(unsafe.Pointer(s))
	sz := 0
	for {
		if strSlice[sz] == 0 {
			break
		}
		sz += stringStride
	}
	str := C.GoStringN((*C.char)(unsafe.Pointer(s)), C.int(sz))
	res, _ := decoder.String(str)
	return TString(res)
}

func stringToC(s TString) C.nar_string_t {
	encoded, _ := encoder.String(string(s) + "\x00")
	cs := C.CBytes([]byte(encoded))
	return C.nar_string_t(unsafe.Pointer(cs))
}

var encoder = func() *encoding.Encoder {
	if goruntime.GOOS == "windows" {
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	} else {
		return utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewEncoder()
	}
}()

//export register_def
func register_def(runtime C.nar_runtime_t, module_name C.nar_string_t, def_name C.nar_string_t, def C.nar_object_t) {
	rt := getRuntime(runtime)
	rt.RegisterDef(bytecode.QualifiedIdentifier(stringToGo(module_name)), ast.Identifier(stringToGo(def_name)), objToGo(def))
}

//export nar_object_kind
func nar_object_kind(_ C.nar_runtime_t, obj C.nar_object_t) C.nar_object_kind_t {
	return C.nar_object_kind_t(objToGo(obj).Kind())
}

//export nar_fail
func nar_fail(_ C.nar_runtime_t, message C.nar_string_t) {
	panic(stringToGo(message))
}

//export nar_apply
func nar_apply(runtime C.nar_runtime_t, fn C.nar_object_t, num_args C.nar_size_t, args *C.nar_object_t) C.nar_object_t {
	rt := getRuntime(runtime)
	args_ := make([]Object, 0, num_args)
	argsSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(args))
	for i := C.nar_size_t(0); i < num_args; i++ {
		args_ = append(args_, objToGo(argsSlice[i]))
	}
	res, err := rt.ApplyFunc(objToGo(fn), args_...)
	if err != nil {
		panic(err)
	}
	return objToC(res)
}

//export nar_unit
func nar_unit(runtime C.nar_runtime_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewUnit())
}

//export nar_char
func nar_char(runtime C.nar_runtime_t, value C.nar_char_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewChar(TChar(value)))
}

//export nar_int
func nar_int(runtime C.nar_runtime_t, value C.nar_int_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewInt(TInt(value)))
}

//export nar_float
func nar_float(runtime C.nar_runtime_t, value C.nar_float_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewFloat(TFloat(value)))
}

//export nar_string
func nar_string(runtime C.nar_runtime_t, value C.nar_string_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewString(stringToGo(value)))
}

//export nar_record
func nar_record(runtime C.nar_runtime_t, size C.nar_size_t, keys *C.nar_string_t, values *C.nar_object_t) C.nar_object_t {
	rt := getRuntime(runtime)
	fields_ := make([]TString, 0, size)
	keys_ := make([]Object, 0, size)
	keysSlice := (*[1 << 30]C.nar_string_t)(unsafe.Pointer(keys))
	valuesSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(values))
	for i := C.nar_size_t(0); i < size; i++ {
		fields_ = append(fields_, stringToGo(keysSlice[i]))
		keys_ = append(keys_, objToGo(valuesSlice[i]))
	}

	return objToC(rt.NewRecord(fields_, keys_))
}

//export nar_list
func nar_list(runtime C.nar_runtime_t, size C.nar_size_t, items *C.nar_object_t) C.nar_object_t {
	rt := getRuntime(runtime)
	items_ := make([]Object, 0, size)
	itemsSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(items))
	for i := C.nar_size_t(0); i < size; i++ {
		items_ = append(items_, objToGo(itemsSlice[i]))
	}
	return objToC(rt.NewList(items_...))
}

//export nar_list_cons
func nar_list_cons(runtime C.nar_runtime_t, head C.nar_object_t, tail C.nar_object_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.newListItem(objToGo(head), objToGo(tail)))
}

//export nar_tuple
func nar_tuple(runtime C.nar_runtime_t, size C.nar_size_t, items *C.nar_object_t) C.nar_object_t {
	rt := getRuntime(runtime)
	items_ := make([]Object, 0, size)
	itemsSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(items))
	for i := C.nar_size_t(0); i < size; i++ {
		items_ = append(items_, objToGo(itemsSlice[i]))
	}
	return objToC(rt.NewTuple(items_...))
}

//export nar_bool
func nar_bool(runtime C.nar_runtime_t, value C.nar_bool_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewBool(value != 0))
}

//export nar_option
func nar_option(runtime C.nar_runtime_t, name C.nar_string_t, size C.nar_size_t, items *C.nar_object_t) C.nar_object_t {
	rt := getRuntime(runtime)
	items_ := make([]Object, 0, size)
	itemsSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(items))
	for i := C.nar_size_t(0); i < size; i++ {
		items_ = append(items_, objToGo(itemsSlice[i]))
	}
	optName := stringToGo(name)
	return objToC(rt.NewOption(optName, items_...))
}

//export nar_func
func nar_func(runtime C.nar_runtime_t, fn C.nar_func_ptr_t, arity C.nar_size_t) C.nar_object_t {
	rt := getRuntime(runtime)
	wrapper, ok := funcWrappers[fn]
	if !ok {
		switch int(arity) {
		case 0:
			{
				wfn := func() Object {
					return objToGo(C.call_func0(C.func0(fn)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 1:
			{
				wfn := func(a Object) Object {
					return objToGo(C.call_func1(C.func1(fn), objToC(a)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 2:
			{
				wfn := func(a Object, b Object) Object {
					return objToGo(C.call_func2(C.func2(fn), objToC(a), objToC(b)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 3:
			{
				wfn := func(a Object, b Object, c Object) Object {
					return objToGo(C.call_func3(C.func3(fn), objToC(a), objToC(b), objToC(c)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 4:
			{
				wfn := func(a Object, b Object, c Object, d Object) Object {
					return objToGo(C.call_func4(C.func4(fn), objToC(a), objToC(b), objToC(c), objToC(d)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 5:
			{
				wfn := func(a Object, b Object, c Object, d Object, e Object) Object {
					return objToGo(C.call_func5(C.func5(fn), objToC(a), objToC(b), objToC(c), objToC(d), objToC(e)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 6:
			{
				wfn := func(a Object, b Object, c Object, d Object, e Object, f Object) Object {
					return objToGo(C.call_func6(C.func6(fn), objToC(a), objToC(b), objToC(c), objToC(d), objToC(e), objToC(f)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 7:
			{
				wfn := func(a Object, b Object, c Object, d Object, e Object, f Object, g Object) Object {
					return objToGo(C.call_func7(C.func7(fn), objToC(a), objToC(b), objToC(c), objToC(d), objToC(e), objToC(f), objToC(g)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		case 8:
			{
				wfn := func(a Object, b Object, c Object, d Object, e Object, f Object, g Object, h Object) Object {
					return objToGo(C.call_func8(C.func8(fn), objToC(a), objToC(b), objToC(c), objToC(d), objToC(e), objToC(f), objToC(g), objToC(h)))
				}
				wrapper = unsafe.Pointer(&wfn)
			}
		}
		funcWrappers[fn] = wrapper
	}
	return objToC(rt.newFunc(wrapper, uint8(arity)))
}

//export nar_native
func nar_native(runtime C.nar_runtime_t, ptr C.nar_native_t) C.nar_object_t {
	rt := getRuntime(runtime)
	return objToC(rt.NewNative(unsafe.Pointer(ptr)))
}

//export nar_to_unit
func nar_to_unit(runtime C.nar_runtime_t, obj C.nar_object_t) {
	rt := getRuntime(runtime)
	_, err := ToUnit(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
}

//export nar_to_char
func nar_to_char(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_char_t {
	rt := getRuntime(runtime)
	v, err := ToChar(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	return C.nar_char_t(v)
}

//export nar_to_int
func nar_to_int(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_int_t {
	rt := getRuntime(runtime)
	v, err := ToInt(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	return C.nar_int_t(v)
}

//export nar_to_float
func nar_to_float(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_float_t {
	rt := getRuntime(runtime)
	v, err := ToFloat(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	return C.nar_float_t(v)
}

//export nar_to_string
func nar_to_string(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_string_t {
	rt := getRuntime(runtime)
	v, err := ToString(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	return stringToC(v)
}

//export nar_to_record
func nar_to_record(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_record_t {
	rt := getRuntime(runtime)
	fields, err := ToRecordFields(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	rec := C.nar_record_t{
		size:   C.nar_size_t(len(fields)),
		keys:   (*C.nar_string_t)(nar_alloc(runtime, C.nar_size_t(len(fields)*C.sizeof_nar_string_t))),
		values: (*C.nar_object_t)(nar_alloc(runtime, C.nar_size_t(len(fields)*C.sizeof_nar_object_t))),
	}
	keysSlice := (*[1 << 30]C.nar_string_t)(unsafe.Pointer(rec.keys))
	valuesSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(rec.values))
	for i, f := range fields {
		keysSlice[i] = stringToC(f.k)
		valuesSlice[i] = objToC(f.v)
	}
	return rec
}

//export nar_to_list
func nar_to_list(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_list_t {
	rt := getRuntime(runtime)
	objects, err := ToList(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	list := C.nar_list_t{
		size:  C.nar_size_t(uint64(len(objects))),
		items: (*C.nar_object_t)(nar_alloc(runtime, C.nar_size_t(len(objects)*C.sizeof_nar_object_t))),
	}
	itemsSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(list.items))
	for i, item := range objects {
		itemsSlice[i] = objToC(item)
	}
	return list
}

//export nar_to_tuple
func nar_to_tuple(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_tuple_t {
	rt := getRuntime(runtime)
	objects, err := ToTuple(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	tuple := C.nar_tuple_t{
		size:  C.nar_size_t(len(objects)),
		items: (*C.nar_object_t)(nar_alloc(runtime, C.nar_size_t(len(objects)*C.sizeof_nar_object_t))),
	}
	itemsSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(tuple.items))
	for i, item := range objects {
		itemsSlice[i] = objToC(item)
	}
	return tuple
}

//export nar_to_bool
func nar_to_bool(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_bool_t {
	rt := getRuntime(runtime)
	v, err := ToBool(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	if v {
		return C.nar_bool_t(1)
	}
	return C.nar_bool_t(0)
}

//export nar_to_option
func nar_to_option(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_option_t {
	rt := getRuntime(runtime)
	n, optValues, err := ToOption(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	optName, err := ToString(rt, n)
	if err != nil {
		panic(err)
	}
	opt := C.nar_option_t{
		name:   stringToC(optName),
		size:   C.nar_size_t(len(optValues)),
		values: (*C.nar_object_t)(nar_alloc(runtime, C.nar_size_t(len(optValues)*C.sizeof_nar_object_t))),
	}
	valuesSlice := (*[1 << 30]C.nar_object_t)(unsafe.Pointer(opt.values))
	for i, item := range optValues {
		valuesSlice[i] = objToC(item)
	}
	return opt
}

//export nar_to_func
func nar_to_func(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_func_t {
	rt := getRuntime(runtime)
	f, a, err := ToFunc(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	return C.nar_func_t{ptr: C.nar_func_ptr_t(f), arity: C.nar_size_t(a)}
}

//export nar_to_native
func nar_to_native(runtime C.nar_runtime_t, obj C.nar_object_t) C.nar_native_t {
	rt := getRuntime(runtime)
	native, err := ToNative(rt, objToGo(obj))
	if err != nil {
		panic(err)
	}
	return C.nar_native_t(native)
}

//export nar_alloc
func nar_alloc(runtime C.nar_runtime_t, size C.nar_size_t) unsafe.Pointer {
	//todo: store pointer and free after frame
	return C.malloc(C.size_t(size))
}

//export nar_print
func nar_print(runtime C.nar_runtime_t, message C.nar_string_t) {
	fmt.Println(stringToGo(message))
}
