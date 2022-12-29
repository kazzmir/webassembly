package data

import (
    "testing"
)

func TestBasic(test *testing.T){
    var x Stack[int]

    x.Push(3)
    x.Push(4)

    if x.Pop() != 4 {
        test.Fatalf("expected 4")
    }

    if x.Pop() != 3 {
        test.Fatalf("expected 3")
    }

    if x.Size() != 0 {
        test.Fatalf("stack should be empty")
    }
}

func TestPushAll(test *testing.T){
    var x Stack[int]
    x.Push(1)
    x.Push(3)
    x.Push(5)

    out := x.PopN(2)
    if out[0] != 3 {
        test.Fatalf("expected 3")
    }
    if out[1] != 5 {
        test.Fatalf("expected 5")
    }

    if x.Size() != 1 {
        test.Fatalf("expected size to be 1")
    }

    x.PushAll([]int{8, 9, 10})

    if x.Size() != 4 {
        test.Fatalf("expected size to be 4")
    }

    if x.Pop() != 10 {
        test.Fatalf("expected 10")
    }

    if x.Pop() != 9 {
        test.Fatalf("expected 9")
    }
    
    if x.Pop() != 8 {
        test.Fatalf("expected 8")
    }
    
    if x.Pop() != 1 {
        test.Fatalf("expected 1")
    }
}
