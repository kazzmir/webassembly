(module
  ;; recursive factorial
  (func (export "fac-rec") (param i64) (result i64)
    (if (result i64) (i64.eq (local.get 0) (i64.const 0))
      (then (i64.const 1))
      (else
        (i64.mul (local.get 0) (call 0 (i64.sub (local.get 0) (i64.const 1))))
        )))

)

(assert_return (invoke "fac-rec" (i64.const 25)) (i64.const 7034535277573963776))

