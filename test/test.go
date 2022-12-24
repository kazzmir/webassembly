package main

import (
    "fmt"
    _ "io"
    "os"
    "strings"
    "sort"
    "path/filepath"
    "github.com/kazzmir/webassembly/lib/core"
)

type SExpression struct {
    Parent *SExpression
    Children []*SExpression
    Value string
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
    
    return strings.Compare(a.Children[0].Value, b.Children[0].Value) == -1
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

func parseSExpression(data string) (SExpression, error) {
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
                current.AddChild(&SExpression{Value: token.Value})
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

func compareSExpression(s1 SExpression, s2 SExpression) bool {
    s1.SortOne()
    s2.SortOne()
    // fmt.Printf("Compare:\n%v\n%v\n", s1, s2)
    return s1.Equal(&s2)
}

func compare(wasmPath string, expectedWatPath string) error {
    module, err := core.Parse(wasmPath)
    if err != nil {
        return err
    }

    output := module.ConvertToWast("")

    // fmt.Printf("Output: %v\n", output)

    sexprActual, err := parseSExpression(output)
    if err != nil {
        return err
    }

    // fmt.Printf("Read s-expr %v\n", sexprActual.String())

    data, err := os.ReadFile(expectedWatPath)
    if err != nil {
        return err
    }
    sexprExpected, err := parseSExpression(string(data))
    if err != nil {
        return err
    }

    if !compareSExpression(sexprActual, sexprExpected) {
        return fmt.Errorf("sexpressions differed")
    }

    return nil
}

func ReplaceExtension(path string, newExt string) string {
    oldExt := filepath.Ext(path)
    base := strings.TrimSuffix(path, oldExt)
    return base + newExt
}

func checkAllTestFiles(){
    paths, err := os.ReadDir("test-files")
    if err != nil {
        fmt.Printf("Error: could not read test-files directory: %v\n", err)
        return
    }

    for _, path := range paths {
        name := path.Name()
        if !path.IsDir() && strings.HasSuffix(name, ".wasm"){
            wasm := filepath.Join("test-files", name)
            expectedWat := filepath.Join("test-files", ReplaceExtension(name, ".wat"))

            err := compare(wasm, expectedWat)
            if err != nil {
                fmt.Printf("Failure: %v vs %v: %v\n", wasm, expectedWat, err)
            }
        }
    }
}

func main() {
    checkAllTestFiles()
}
