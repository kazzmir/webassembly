(module
  (type (;0;) (func (result i32)))
  (type (;1;) (func (result i32)))
  (type (;2;) (func (param i32) (result i32)))
  (func (;0;) (type 0) (result i32)
    i32.const 42)
  (func (;1;) (type 0) (result i32)
    i32.const 13)
  (func (;2;) (type 2) (param i32) (result i32)
    local.get 0
    call_indirect (type 1))
  (table (;0;) 2 funcref)
  (export "callByIndex" (func 2))
  (elem (;0;) (i32.const 0) func 0 1))
