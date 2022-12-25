(module
  (type (;0;) (func (param i32 i32)))
  (type (;1;) (func))
  (import "console" "log" (func (;0;) (type 0)))
  (import "js" "mem" (memory (;0;) 1))
  (func (;1;) (type 1)
    i32.const 0
    i32.const 2
    call 0)
  (export "writeHi" (func 1))
  (data (;0;) (i32.const 0) "Hi"))
