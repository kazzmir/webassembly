(module
  (func (export "local1") (param i32) (result i32)
    (block (local.get 0))
  )
)

(assert_return (invoke "local1" (i32.const 3)) (i32.const 3))
