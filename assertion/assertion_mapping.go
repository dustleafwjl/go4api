/*
 * go4api - a api testing tool written in Go
 * Created by: Ping Zhu 2018
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package assertion

import (
    
)

type AssertionDetails struct {
    AssertionKey string
    AssertionFunc interface{}  // function
    ApplyTypes []string        // [string, number, bool]
}

var assertionMapping = make(map[string]*AssertionDetails)

func init() {
    assertionMapping["Equals"] = &AssertionDetails{"Equals", Equals, []string{"string", "number", "float64", "bool", "true", "false"}}

    assertionMapping["Contains"] = &AssertionDetails{"Contains", Contains, []string{"string"}}
    assertionMapping["StartsWith"] = &AssertionDetails{"StartsWith", StartsWith, []string{"string"}}
    assertionMapping["EndsWith"] = &AssertionDetails{"EndsWith", EndsWith, []string{"string"}}

    assertionMapping["NotEquals"] = &AssertionDetails{"NotEquals", NotEquals, []string{"string", "number", "float64", "bool", "true", "false"}}

    assertionMapping["Less"] = &AssertionDetails{"Less", Less, []string{"float64", "number"}}
    assertionMapping["LessOrEquals"] = &AssertionDetails{"LessOrEquals", LessOrEquals, []string{"float64", "number"}}
    assertionMapping["Greater"] = &AssertionDetails{"Greater", Greater, []string{"float64", "number"}}
    assertionMapping["GreaterOrEquals"] = &AssertionDetails{"GreaterOrEquals", GreaterOrEquals, []string{"float64", "number"}}

    assertionMapping["Match"] = &AssertionDetails{"Match", Match, []string{"string"}}
}

// To support assertion here:
// if response body is xml: [key, using xpath] [operator, like Equals, ...] [value, can use regrex]
// if response body is html: [key, using xpath, css] [operator, like Equals, ...] [value, can use regrex]
// if response body is json: [key] [operator, like Equals, ...] [value, can use regrex]

// for String:
// Equals
// Contains
// StartsWith
// EndsWith

// for Numeric:
// Equals
// NotEquals
// Less
// LessOrEquals
// Greater
// GreaterOrEquals

// for Bool (true, false):
// Equals
// NotEquals

// for general regrex
// Match





