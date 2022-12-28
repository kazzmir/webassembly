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

  (func (export "as-br_if-cond")
    (block (br_if 0 (br_if 0 (i32.const 1) (i32.const 1))))
  )

  (func (export "as-br_if-value") (result i32)
    (block (result i32)
    (drop (br_if 0 (br_if 0 (i32.const 1) (i32.const 2)) (i32.const 3)))
     (i32.const 4)))

  (func (export "as-br_if-value-cond") (param i32) (result i32)
    (block (result i32)
      (drop (br_if 0 (i32.const 2) (br_if 0 (i32.const 1) (local.get 0))))
      (i32.const 4)))

  (func (export "as-br_table-index")
    (block (br_table 0 0 0 (br_if 0 (i32.const 1) (i32.const 2)))))

  (func (export "as-br_table-value") (result i32)
    (block (result i32)
      (br_table 0 0 0
                (br_if 0 (i32.const 1) (i32.const 2))
                (i32.const 3))
      (i32.const 4)
      ))

  (func (export "as-br_table-value-index") (result i32)
     (block (result i32)
       (br_table 0 0
                 (i32.const 2)
                 (br_if 0 (i32.const 1) (i32.const 3)))
       (i32.const 4)))

  (func (export "as-return-value") (result i64)
    (block (result i64) (return (br_if 0 (i64.const 1) (i32.const 2)))))

  (func (export "as-if-cond") (param i32) (result i32)
    (block (result i32)
      (if (result i32)
        (br_if 0 (i32.const 1) (local.get 0))
        (then (i32.const 2))
        (else (i32.const 3)))))

  (func (export "as-if-then") (param i32 i32)
     (block
       (if (local.get 0) (then (br_if 1 (local.get 1))) (else (call $dummy)))))

  (func (export "as-if-else") (param i32 i32)
     (block
       (if (local.get 0) (then (call $dummy)) (else (br_if 1 (local.get 1))))))

  (func (export "as-select-first") (param i32) (result i32)
     (block (result i32)
       (select
         (br_if 0 (i32.const 3) (i32.const 10))
         (i32.const 2)
         (local.get 0))))

  (func (export "as-select-second") (param i32) (result i32)
    (block (result i32)
       (select (i32.const 1) (br_if 0 (i32.const 3) (i32.const 10)) (local.get 0))))

  (func (export "as-select-cond") (result i32)
     (block (result i32)
        (select (i32.const 1) (i32.const 2) (br_if 0 (i32.const 3) (i32.const 10)))))

  (func $f (param i32 i32 i32) (result i32) (i32.const -1))
  (func (export "as-call-first") (result i32)
    (block (result i32)
      (call $f
        (br_if 0 (i32.const 12) (i32.const 1))
        (i32.const 2)
        (i32.const 3))))

  (func (export "as-call-mid") (result i32)
     (block (result i32)
       (call $f
         (i32.const 1)
         (br_if 0 (i32.const 13) (i32.const 1))
         (i32.const 3))))

  (func (export "as-call-last") (result i32)
     (block (result i32)
       (call $f
         (i32.const 1)
         (i32.const 2)
         (br_if 0 (i32.const 14) (i32.const 1)))))

  (func $func (param i32 i32 i32) (result i32) (local.get 0))
  (type $check (func (param i32 i32 i32) (result i32)))
  (table funcref (elem $func))

  (func (export "as-call_indirect-func") (result i32)
     (block (result i32)
       (call_indirect (type $check)
         (br_if 0 (i32.const 4) (i32.const 10))
         (i32.const 1) (i32.const 2) (i32.const 0))))

  (func (export "as-call_indirect-first") (result i32)
     (block (result i32)
       (call_indirect (type $check)
          (i32.const 1) (br_if 0 (i32.const 4) (i32.const 10)) (i32.const 2) (i32.const 0))))

  (func (export "as-call_indirect-mid") (result i32)
     (block (result i32)
         (call_indirect (type $check)
            (i32.const 1) (i32.const 2) (br_if 0 (i32.const 4) (i32.const 10)) (i32.const 0))))

  (func (export "as-call_indirect-last") (result i32)
      (block (result i32)
         (call_indirect (type $check)
            (i32.const 1) (i32.const 2) (i32.const 3) (br_if 0 (i32.const 4) (i32.const 10)))))

  (func (export "as-local.set-value") (param i32) (result i32)
     (local i32)
     (block (result i32)
       (local.set 0 (br_if 0 (i32.const 17) (local.get 0)))
       (i32.const -1)))

  (func (export "as-local.tee-value") (param i32) (result i32)
     (block (result i32)
       (local.tee 0 (br_if 0 (i32.const 1) (local.get 0)))
       (return (i32.const -1))))

  (global $a (mut i32) (i32.const 10))
  (func (export "as-global.set-value") (param i32) (result i32)
     (block (result i32)
        (global.set $a (br_if 0 (i32.const 1) (local.get 0)))
        (return (i32.const -1))))

  (memory 1)
  (func (export "as-load-address") (result i32)
    (block (result i32) (i32.load (br_if 0 (i32.const 1) (i32.const 1)))))

  (func (export "as-loadN-address") (result i32)
    (block (result i32) (i32.load8_s (br_if 0 (i32.const 30) (i32.const 1)))))

  (func (export "as-store-address") (result i32)
    (block (result i32)
      (i32.store (br_if 0 (i32.const 30) (i32.const 1)) (i32.const 7)) (i32.const -1)))

  (func (export "as-store-value") (result i32)
    (block (result i32)
      (i32.store (i32.const 2) (br_if 0 (i32.const 31) (i32.const 1))) (i32.const -1)))

  (func (export "as-storeN-address") (result i32)
    (block (result i32)
      (i32.store8 (br_if 0 (i32.const 32) (i32.const 1)) (i32.const 7)) (i32.const -1)))

  (func (export "as-storeN-value") (result i32)
    (block (result i32)
      (i32.store16 (i32.const 2) (br_if 0 (i32.const 33) (i32.const 1))) (i32.const -1)))

  (func (export "as-unary-operand") (result f64)
    (block (result f64) (f64.neg (br_if 0 (f64.const 1.0) (i32.const 1)))))

  (func (export "as-binary-left") (result i32)
     (block (result i32) (i32.add (br_if 0 (i32.const 1) (i32.const 1)) (i32.const 10))))

  (func (export "as-binary-right") (result i32)
    (block (result i32) (i32.sub (i32.const 10) (br_if 0 (i32.const 1) (i32.const 1)))))

  (func (export "as-test-operand") (result i32)
    (block (result i32) (i32.eqz (br_if 0 (i32.const 0) (i32.const 1)))))

  (func (export "as-compare-left") (result i32)
    (block (result i32) (i32.le_u (br_if 0 (i32.const 1) (i32.const 1)) (i32.const 10))))

  (func (export "as-compare-right") (result i32)
    (block (result i32) (i32.ne (i32.const 10) (br_if 0 (i32.const 1) (i32.const 42)))))

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

(assert_return (invoke "as-br_if-cond"))
(assert_return (invoke "as-br_if-value") (i32.const 1))
(assert_return (invoke "as-br_if-value-cond" (i32.const 0)) (i32.const 2))
(assert_return (invoke "as-br_if-value-cond" (i32.const 1)) (i32.const 1))

(assert_return (invoke "as-br_table-index"))
(assert_return (invoke "as-br_table-value") (i32.const 1))
(assert_return (invoke "as-br_table-value-index") (i32.const 1))

(assert_return (invoke "as-return-value") (i64.const 1))

(assert_return (invoke "as-if-cond" (i32.const 0)) (i32.const 2))
(assert_return (invoke "as-if-cond" (i32.const 1)) (i32.const 1))

(assert_return (invoke "as-if-then" (i32.const 0) (i32.const 0)))
(assert_return (invoke "as-if-then" (i32.const 4) (i32.const 0)))
(assert_return (invoke "as-if-then" (i32.const 0) (i32.const 1)))
(assert_return (invoke "as-if-then" (i32.const 4) (i32.const 1)))

(assert_return (invoke "as-if-else" (i32.const 0) (i32.const 0)))
(assert_return (invoke "as-if-else" (i32.const 3) (i32.const 0)))
(assert_return (invoke "as-if-else" (i32.const 0) (i32.const 1)))
(assert_return (invoke "as-if-else" (i32.const 3) (i32.const 1)))

(assert_return (invoke "as-select-first" (i32.const 0)) (i32.const 3))
(assert_return (invoke "as-select-first" (i32.const 1)) (i32.const 3))

(assert_return (invoke "as-select-second" (i32.const 0)) (i32.const 3))
(assert_return (invoke "as-select-second" (i32.const 1)) (i32.const 3))
(assert_return (invoke "as-select-cond") (i32.const 3))

(assert_return (invoke "as-call-first") (i32.const 12))
(assert_return (invoke "as-call-mid") (i32.const 13))
(assert_return (invoke "as-call-last") (i32.const 14))

(assert_return (invoke "as-call_indirect-func") (i32.const 4))
(assert_return (invoke "as-call_indirect-first") (i32.const 4))
(assert_return (invoke "as-call_indirect-mid") (i32.const 4))
(assert_return (invoke "as-call_indirect-last") (i32.const 4))

(assert_return (invoke "as-local.set-value" (i32.const 0)) (i32.const -1))
(assert_return (invoke "as-local.set-value" (i32.const 1)) (i32.const 17))

(assert_return (invoke "as-local.tee-value" (i32.const 0)) (i32.const -1))
(assert_return (invoke "as-local.tee-value" (i32.const 1)) (i32.const 1))

(assert_return (invoke "as-global.set-value" (i32.const 0)) (i32.const -1))
(assert_return (invoke "as-global.set-value" (i32.const 1)) (i32.const 1))

(assert_return (invoke "as-load-address") (i32.const 1))
(assert_return (invoke "as-loadN-address") (i32.const 30))

(assert_return (invoke "as-store-address") (i32.const 30))
(assert_return (invoke "as-store-value") (i32.const 31))
(assert_return (invoke "as-storeN-address") (i32.const 32))
(assert_return (invoke "as-storeN-value") (i32.const 33))
(assert_return (invoke "as-unary-operand") (f64.const 1.0))

(assert_return (invoke "as-binary-left") (i32.const 1))
(assert_return (invoke "as-binary-right") (i32.const 1))
(assert_return (invoke "as-test-operand") (i32.const 0))
(assert_return (invoke "as-compare-left") (i32.const 1))
(assert_return (invoke "as-compare-right") (i32.const 1))
