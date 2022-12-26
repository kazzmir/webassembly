(module
  (func (export "xyz") (result i32)
    ;; (i32.const 4)
    (block (result i32)
      (block (result i32)
        (i32.const 28) ;; gets dropped due to br
        (i32.const 10) ;; gets dropped due to br
        (i32.const 3)
        (br 1)))))

(assert_return (invoke "xyz") (i32.const 3))
