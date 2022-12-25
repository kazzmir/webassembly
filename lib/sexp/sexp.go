package sexp

import (
    "fmt"
    "sort"
    "strings"
)

type SExpression struct {
    Parent *SExpression
    Children []*SExpression
    Name string // (xyz sub1 sub2 ...) the name is 'xyz'
    Value string // if no parens, then this is just the token
}

type ByFirst []*SExpression

func (first ByFirst) Len() int {
    return len(first)
}

func (first ByFirst) Swap(i, j int) {
    first[i], first[j] = first[j], first[i]
}

func (first ByFirst) Less(i, j int) bool {
    a := first[i]
    b := first[j]
    
    return strings.Compare(a.Name, b.Name) == -1
}

/* sorts only the immediate children */
func (sexpr *SExpression) SortOne(){
    sort.Stable(ByFirst(sexpr.Children[1:len(sexpr.Children)]))
}

func (sexpr *SExpression) Equal(other *SExpression) bool {
    if sexpr.Value != "" || other.Value != "" {
        return sexpr.Value == other.Value
    }
    
    if len(sexpr.Children) != len(other.Children) {
        return false
    }

    for i := 0; i < len(sexpr.Children); i++ {
        if !sexpr.Children[i].Equal(other.Children[i]) {
            return false
        }
    }

    return true
}

func (sexpr *SExpression) String() string {
    if sexpr.Value != "" {
        return sexpr.Value
    }

    var out strings.Builder
    out.WriteByte('(')
    for i, child := range sexpr.Children {
        out.WriteString(child.String())
        if i < len(sexpr.Children) - 1 {
            out.WriteByte(' ')
        }
    }
    out.WriteByte(')')

    return out.String()
}

func (sexpr *SExpression) AddChild(child *SExpression){
    child.Parent = sexpr
    sexpr.Children = append(sexpr.Children, child)
}

const (
    TokenEOF = iota
    TokenLeftParens
    TokenRightParens
    TokenData
)

type Token struct {
    Value string
    Kind int
}

func nextToken(reader *strings.Reader) Token {
    for {
        first, err := reader.ReadByte()
        if err != nil {
            return Token{Kind: TokenEOF}
        }
        if first == ' ' || first == '\n' || first == '\t' {
            continue
        }
        if first == '(' {
            return Token{Kind: TokenLeftParens, Value: "("}
        }
        if first == ')' {
            return Token{Kind: TokenRightParens, Value: ")"}
        }

        // look for a line starting with ;;
        // and if found, read until the end of the line
        if first == ';' {
            next, err := reader.ReadByte()
            if err != nil {
                reader.UnreadByte()
            } else {
                if next == ';' {
                    for {
                        next, err = reader.ReadByte()
                        if err != nil {
                            break
                        }
                        if next == '\n' {
                            break
                        }
                    }
                    continue
                } else {
                    reader.UnreadByte()
                }
            }
        }
        
        var out strings.Builder
        out.WriteByte(first)
        for {
            next, err := reader.ReadByte()
            if err != nil {
                reader.UnreadByte()
                break
            }

            if next == ' ' || next == '\n' || next == '\t' {
                break
            }

            if next == '(' || next == ')' {
                reader.UnreadByte()
                break
            } else {
                out.WriteByte(next)
            }
        }
        return Token{Kind: TokenData, Value: out.String()}
    }
}

func ParseSExpression(data string) (SExpression, error) {
    reader := strings.NewReader(data)

    var top *SExpression
    var current *SExpression
    
    quit := false
    for !quit {
        token := nextToken(reader)
        // fmt.Printf("Next token: %+v\n", token)
        switch token.Kind {
            case TokenLeftParens:
                if top == nil {
                    top = new(SExpression)
                    current = top
                } else {
                    next := new(SExpression)
                    current.AddChild(next)
                    current = next
                }
            case TokenRightParens:
                current = current.Parent
            case TokenData:
                if current.Name == "" {
                    current.Name = token.Value
                } else {
                    current.AddChild(&SExpression{Value: token.Value})
                }
            case TokenEOF:
                quit = true
        }
    }

    if current != nil {
        return SExpression{}, fmt.Errorf("did not parse all parens")
    }

    if nextToken(reader).Kind != TokenEOF {
        return SExpression{}, fmt.Errorf("unable to parse")
    }

    return *top, nil
}

