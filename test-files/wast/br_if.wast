;; Test `br_if` operator

(module
  (func $dummy)

  (func (export "type-i32")
    (block (drop (i32.ctz (br_if 0 (i32.const 0) (i32.const 1)))))
  )
  (func (export "type-i64")
    (block (drop (i64.ctz (br_if 0 (i64.const 0) (i32.const 1)))))
  )
  (func (export "type-f32")
    (block (drop (f32.neg (br_if 0 (f32.const 0) (i32.const 1)))))
  )
  (func (export "type-f64")
    (block (drop (f64.neg (br_if 0 (f64.const 0) (i32.const 1)))))
  )

  (func (export "type-i32-value") (result i32)
    (block (result i32) (i32.ctz (br_if 0 (i32.const 1) (i32.const 1))))
  )
  (func (export "type-i64-value") (result i64)
    (block (result i64) (i64.ctz (br_if 0 (i64.const 2) (i32.const 1))))
  )
  (func (export "type-f32-value") (result f32)
    (block (result f32) (f32.neg (br_if 0 (f32.const 3) (i32.const 1))))
  )
  (func (export "type-f64-value") (result f64)
    (block (result f64) (f64.neg (br_if 0 (f64.const 4) (i32.const 1))))
  )
)

(assert_return (invoke "type-i32"))
(assert_return (invoke "type-i64"))

(assert_return (invoke "type-f32"))
(assert_return (invoke "type-f64"))

(assert_return (invoke "type-i32-value") (i32.const 1))
(assert_return (invoke "type-i64-value") (i64.const 2))
(assert_return (invoke "type-f32-value") (f32.const 3))
(assert_return (invoke "type-f64-value") (f64.const 4))
