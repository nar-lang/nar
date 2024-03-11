#include <stdlib.h>
#include "_cgo_export.h"

int init_wrapper(void* init_fn, nar_runtime_t runtime) {
    init_data_t *data = (init_data_t *)malloc(sizeof(init_data_t));
    data->runtime = runtime;

    data->register_def = &register_def;
    data->nar_object_kind = &nar_object_kind;
    data->nar_fail = &nar_fail;
    data->nar_apply = &nar_apply;
    data->nar_alloc = &nar_alloc;
    data->nar_print = &nar_print;

    data->nar_unit = &nar_unit;
    data->nar_char = &nar_char;
    data->nar_int = &nar_int;
    data->nar_float = &nar_float;
    data->nar_string = &nar_string;
    data->nar_record = &nar_record;
    data->nar_list = &nar_list;
    data->nar_list_cons = &nar_list_cons;
    data->nar_tuple = &nar_tuple;
    data->nar_bool = &nar_bool;
    data->nar_option = &nar_option;
    data->nar_func = &nar_func;
    data->nar_native = &nar_native;

    data->nar_to_unit = &nar_to_unit;
    data->nar_to_char = &nar_to_char;
    data->nar_to_int = &nar_to_int;
    data->nar_to_float = &nar_to_float;
    data->nar_to_string = &nar_to_string;
    data->nar_to_record = &nar_to_record;
    data->nar_to_list = &nar_to_list;
    data->nar_to_tuple = &nar_to_tuple;
    data->nar_to_bool = &nar_to_bool;
    data->nar_to_option = &nar_to_option;
    data->nar_to_func = &nar_to_func;
    data->nar_to_native = &nar_to_native;

    (*(init_fn_t)init_fn)(data);

    return 0;
}

nar_object_t call_func0(func0 fn) {
	return fn();
}

nar_object_t call_func1(func1 fn, nar_object_t a) {
	return fn(a);
}

nar_object_t call_func2(func2 fn, nar_object_t a, nar_object_t b) {
	return fn(a, b);
}

nar_object_t call_func3(func3 fn, nar_object_t a, nar_object_t b, nar_object_t c) {
	return fn(a, b, c);
}

nar_object_t call_func4(func4 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d) {
	return fn(a, b, c, d);
}

nar_object_t call_func5(func5 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e) {
	return fn(a, b, c, d, e);
}

nar_object_t call_func6(func6 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e, nar_object_t f) {
	return fn(a, b, c, d, e, f);
}

nar_object_t call_func7(func7 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e, nar_object_t f, nar_object_t g) {
	return fn(a, b, c, d, e, f, g);
}

nar_object_t call_func8(func8 fn, nar_object_t a, nar_object_t b, nar_object_t c, nar_object_t d, nar_object_t e, nar_object_t f, nar_object_t g, nar_object_t h) {
	return fn(a, b, c, d, e, f, g, h);
}
