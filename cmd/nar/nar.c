#include <nar.h>

extern void goStdOut(nar_runtime_t rt, nar_cstring_t msg);

void goStdoutWrapper(nar_runtime_t rt, nar_cstring_t msg) {
  goStdOut(rt, msg);
}
