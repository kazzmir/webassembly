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

  (func (export "as-block-first") (param i32) (result i32)
    (block
      (br_if 0 (local.get 0))
      (return (i32.const 2)))
    (i32.const 3))


  (func (export "as-block-mid") (param i32) (result i32)
     (block (call $dummy) (br_if 0 (local.get 0)) (return (i32.const 2)))
       (i32.const 3))

  (func (export "as-block-last") (param i32)
     (block (call $dummy) (call $dummy) (br_if 0 (local.get 0))))

  (func (export "as-block-first-value") (param i32) (result i32)
     (block (result i32)
       (drop (br_if 0 (i32.const 10) (local.get 0))) (return (i32.const 11))))

  (func (export "as-block-mid-value") (param i32) (result i32)
     (block (result i32)
       (call $dummy)
       (drop (br_if 0 (i32.const 20) (local.get 0)))
       (return (i32.const 21))))

  (func (export "as-block-last-value") (param i32) (result i32)
      (block (result i32)
      (call $dummy) (call $dummy) (br_if 0 (i32.const 11) (local.get 0))))

  (func (export "as-loop-first") (param i32) (result i32)
      (block (loop (br_if 1 (local.get 0)) (return (i32.const 2)))) (i32.const 3)
  )

  (func (export "as-loop-mid") (param i32) (result i32)
    (block (loop (call $dummy) (br_if 1 (local.get 0)) (return (i32.const 2))))
    (i32.const 4))

  (func (export "as-loop-last") (param i32)
    (loop (call $dummy) (br_if 1 (local.get 0))))

  (func (export "as-br-value") (result i32)
    (block (result i32) (br 0 (br_if 0 (i32.const 1) (i32.const 2)))))

)

(assert_return (invoke "type-i32"))
(assert_return (invoke "type-i64"))

(assert_return (invoke "type-f32"))
(assert_return (invoke "type-f64"))

(assert_return (invoke "type-i32-value") (i32.const 1))
(assert_return (invoke "type-i64-value") (i64.const 2))
(assert_return (invoke "type-f32-value") (f32.const 3))
(assert_return (invoke "type-f64-value") (f64.const 4))

(assert_return (invoke "as-block-first" (i32.const 0)) (i32.const 2))
(assert_return (invoke "as-block-first" (i32.const 1)) (i32.const 3))
(assert_return (invoke "as-block-mid" (i32.const 0)) (i32.const 2))
(assert_return (invoke "as-block-mid" (i32.const 1)) (i32.const 3))
(assert_return (invoke "as-block-last" (i32.const 0)))
(assert_return (invoke "as-block-last" (i32.const 1)))

(assert_return (invoke "as-block-first-value" (i32.const 0)) (i32.const 11))
(assert_return (invoke "as-block-first-value" (i32.const 1)) (i32.const 10))
(assert_return (invoke "as-block-mid-value" (i32.const 0)) (i32.const 21))
(assert_return (invoke "as-block-mid-value" (i32.const 1)) (i32.const 20))
(assert_return (invoke "as-block-last-value" (i32.const 0)) (i32.const 11))
(assert_return (invoke "as-block-last-value" (i32.const 1)) (i32.const 11))

(assert_return (invoke "as-loop-first" (i32.const 0)) (i32.const 2))
(assert_return (invoke "as-loop-first" (i32.const 1)) (i32.const 3))

(assert_return (invoke "as-loop-mid" (i32.const 0)) (i32.const 2))
(assert_return (invoke "as-loop-mid" (i32.const 1)) (i32.const 4))
(assert_return (invoke "as-loop-last" (i32.const 0)))
(assert_return (invoke "as-loop-last" (i32.const 1)))

(assert_return (invoke "as-br-value") (i32.const 1))
