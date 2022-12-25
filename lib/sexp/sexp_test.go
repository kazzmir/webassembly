package sexp

import (
    "testing"
)

func TestBasic(test *testing.T){
    value, err := ParseSExpression("(x)")
    if err != nil {
        test.Fatalf("Could not parse (x): %v", err)
    }

    if value.Value != "" {
        test.Fatalf("expected top level sexp to not be a value, but was %v", value.Value)
    }

    if value.Name != "x" {
        test.Fatalf("expected top level sexp to be named 'x' but was '%v'", value.Name)
    }

    if value.Parent != nil {
        test.Fatalf("expected top level sexp to have no parent, but was %v", value.Parent)
    }

    if len(value.Children) != 0 {
        test.Fatalf("expected (x) to have zero child, but had %v", len(value.Children))
    }
}

func TestMore(test *testing.T){
    input := "(x a (b 1 2) c)"
    value, err := ParseSExpression(input)
    if err != nil {
        test.Fatalf("Could not parse '%v': %v", input, err)
    }

    if value.Name != "x" {
        test.Fatalf("unexpected name %v", value.Name)
    }
    
    if len(value.Children) != 3 {
        test.Fatalf("expected 3 children for x but was %v", len(value.Children))
    }

    childA := value.Children[0]
    if childA.Value != "a" {
        test.Fatalf("expected first child to be the value 'a' but was '%v'", childA.Value)
    }

    childB := value.Children[1]
    if childB.Name != "b" {
        test.Fatalf("expected 'b' child to have name 'b' but was '%v'", childB.Name)
    }
    
    if len(childB.Children) != 2 {
        test.Fatalf("expected b node to have 2 children but had %v", len(childB.Children))
    }

    childC := value.Children[2]
    if childC.Value != "c" {
        test.Fatalf("expected 'c' child to have value 'c' but was '%v'", childC.Value)
    }
}

func TestComments(test *testing.T){
    input := `
;; ignore this
(x a
;; more ignore
b c)
`
    value, err := ParseSExpression(input)
    if err != nil {
        test.Fatalf("Could not parse '%v': %v", input, err)
    }

    if value.Name != "x" {
        test.Fatalf("unexpectedf name %v", value.Name)
    }

    if len(value.Children) != 3 {
        test.Fatalf("expected 3 children for x but was %v", len(value.Children))
    }
}
