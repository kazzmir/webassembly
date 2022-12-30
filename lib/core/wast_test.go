package core

import (
    "testing"
    "math"
)

func near(a float64, b float64) bool {
    if math.Abs(a - b) > 0.00000001 {
        return false
    }

    return true
}

func TestParseFloat(test *testing.T){
    value, err := parseFloat32("0x0p+0")
    if err != nil {
        test.Fatalf("unable to parse: %v", err)
    }
    if !near(float64(value), 0) {
        test.Fatalf("value was not near 0: %v", value)
    }

    value, err = parseFloat32("0x3p+2")
    if err != nil {
        test.Fatalf("unable to parse: %v", err)
    }

    if !near(float64(value), 12) {
        test.Fatalf("value was not near 12: %v", value)
    }
}
