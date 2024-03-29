(module

  (func $test (param i32) (result i32)
    (i32.add (i32.const 3) (local.get 0)))

  (func (export "local1") (param i32) (result i32)
    (block (result i32) (local.get 0))
  )

  (func (export "local-call") (result i32)
    (block (result i32)
      (i32.const 4)
      (call $test)))

  (func (export "local-param") (param i32) (result i32)
     (local i32)
     (local.get 1))
)

(assert_return (invoke "local1" (i32.const 3)) (i32.const 3))
(assert_return (invoke "local-call") (i32.const 7))
(assert_return (invoke "local-param" (i32.const 8)) (i32.const 0))
