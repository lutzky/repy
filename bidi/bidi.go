// Package bidi includes basic tooling for working with bidirectional
// text.
//
// We really wish we could use preexisting code for this, but
// https://godoc.org/golang.org/x/text/unicode/bidi is under construction.
package bidi

import (
	"strings"
	"unicode"
)

const neutralRunes = "-./"

// Reverse reverses a visual-Hebrew string into logical order.
func Reverse(s string) string {
	runes := []rune(s)

	mainStack := newRuneStack(len(s))
	numStack := newRuneStack(len(s))

	numberRun := false
	for i, r := range runes {
		wasNumberRun := numberRun
		if unicode.IsNumber(r) {
			// Number - start/continue number run
			numberRun = true
		} else if strings.ContainsRune(neutralRunes, r) {
			// Neutral - continue number run iff we're in a middle of a run with
			// only numbers and neutrals.
			if numberRun {
				numberRun = false
				for j := i + 1; j < len(runes); j++ {
					if unicode.IsNumber(runes[j]) {
						numberRun = true
					}
				}
			}
		} else {
			// Not a number - end number run
			numberRun = false
		}

		if wasNumberRun && !numberRun {
			emptyInto(&numStack, &mainStack)
		}

		if numberRun {
			numStack.push(runes[i])
		} else {
			switch runes[i] {
			case '(':
				mainStack.push(')')
			case ')':
				mainStack.push('(')
			default:
				mainStack.push(runes[i])
			}
		}
	}

	emptyInto(&numStack, &mainStack)
	return mainStack.toRevString()
}

type runeStack []rune

func newRuneStack(size int) runeStack {
	return make([]rune, 0, size)
}

func (rs *runeStack) push(r rune) {
	*rs = append(*rs, r)
}

func (rs *runeStack) pop() rune {
	n := len(*rs) - 1
	result := (*rs)[n]
	*rs = (*rs)[:n]
	return result
}

func emptyInto(src, dest *runeStack) {
	for len(*src) > 0 {
		dest.push(src.pop())
	}
}

func (rs runeStack) toRevString() string {
	n := len(rs)
	runes := make([]rune, n)
	for i := 0; i < n; i++ {
		runes[i] = rs[n-i-1]
	}
	return string(runes)
}
