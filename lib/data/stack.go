package data

type Stack[T comparable] struct {
    Values []T
}

func (stack *Stack[T]) Size() int {
    return len(stack.Values)
}

func (stack *Stack[T]) Get(depth uint32) T {
    return stack.Values[len(stack.Values) - int(depth) - 1]
}

func (stack *Stack[T]) PopN(n int) []T {
    var out []T

    for i := 0; i < n; i++ {
        out = append(out, stack.Values[len(stack.Values)-n+i])
    }
    stack.Values = stack.Values[0:len(stack.Values)-n]

    return out
}

func (stack *Stack[T]) Push(value T){
    stack.Values = append(stack.Values, value)
}

func (stack *Stack[T]) PushAll(values []T){
    for _, value := range values {
        stack.Push(value)
    }
}

func (stack *Stack[T]) ToArray() []T {
    var out []T
    /* FIXME: use copy */
    for _, value := range stack.Values {
        out = append(out, value)
    }
    return out
}

/* returns the first index of 'value' starting from the top (0) */
func (stack *Stack[T]) Find(value T) (int, bool) {
    for i := 0; i < len(stack.Values); i++ {
        if stack.Values[len(stack.Values)-1-i] == value {
            return i, true
        }
    }

    return 0, false
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

/* chop off the last couple of elements such that the stack has `size' elements in it */
func (stack *Stack[T]) Reduce(size int){
    stack.Values = stack.Values[0:min(size, len(stack.Values))]
}

func (stack *Stack[T]) Pop() T {
    if len(stack.Values) == 0 {
        var x T
        return x
    }

    t := stack.Values[len(stack.Values)-1]
    stack.Values = stack.Values[0:len(stack.Values)-1]
    return t
}

