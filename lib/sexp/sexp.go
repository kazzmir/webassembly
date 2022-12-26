package sexp

/* Parse .wast style sexpressions. These are slightly restricted sexpressions in that
 * every sexpression always starts with some name, such as (x a b). The first thing
 * in an sexpression will never be another sexpression, such as ((f) b c)
 */

import (
    "fmt"
    "io"
    "bufio"
    "os"
    "errors"
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
    out.WriteString(sexpr.Name)
    for _, child := range sexpr.Children {
        out.WriteByte(' ')
        out.WriteString(child.String())
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

func nextToken(reader io.ByteScanner) Token {
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

var MismatchedParensError = errors.New("unmatched right parens")

func ParseSExpressionReader(reader io.ByteScanner) (SExpression, error) {
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
                /* if we reached the end of the current sexp then quit */
                if current == nil {
                    quit = true
                }
            case TokenData:
                if current == nil {
                    return SExpression{}, fmt.Errorf("no left parens seen")
                }
                if current.Name == "" {
                    current.Name = token.Value
                } else {
                    current.AddChild(&SExpression{Value: token.Value})
                }
            case TokenEOF:
                quit = true
        }
    }

    if top == nil {
        return SExpression{}, io.EOF
    }

    if current != nil {
        return SExpression{}, MismatchedParensError
    }

    /* there might be more sexps after this one so don't touch the reader */

    return *top, nil
}

func ParseSExpression(data string) (SExpression, error) {
    return ParseSExpressionReader(strings.NewReader(data))
}

func ParseSExpressionFile(path string) (SExpression, error) {
    file, err := os.Open(path)
    if err != nil {
        return SExpression{}, err
    }
    defer file.Close()

    return ParseSExpressionReader(bufio.NewReader(file))
}
