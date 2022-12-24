(module
  (type (;0;) (func (param i32 i32) (result i32)))
  (import "js" "mem" (memory (;0;) 1))
  (func (;0;) (type 0) (param i32 i32) (result i32)
    (local i32 i32)
    local.get 0
    local.get 1
    i32.const 4
    i32.mul
    i32.add
    local.set 2
    block  ;; label = @1
      loop  ;; label = @2
        local.get 0
        local.get 2
        i32.eq
        br_if 1 (;@1;)
        local.get 3
        local.get 0
        i32.load
        i32.add
        local.set 3
        local.get 0
        i32.const 4
        i32.add
        local.set 0
        br 0 (;@2;)
      end
    end
    local.get 3)
  (export "accumulate" (func 0)))
