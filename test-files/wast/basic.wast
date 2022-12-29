(module
  ;; return two thing on the stack
  (func $two (result i32 i32)
    (i32.const 8)
    (return (i32.const 1) (i32.const 2)))

  (func (export "check") (result i32)
    ;; this call should put two things on the stack
    (call $two)
    ;; add them together
    (i32.add))
)

(assert_return (invoke "check") (i32.const 3))
