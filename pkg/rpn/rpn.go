package rpn

import (
	"errors"
	"strconv"
	"strings"
)

func Calc(expression string) (float64, error) {
	var nums []float64
	var ops []rune
	var currNumStr strings.Builder

	var err error
	for _, char := range expression {
		if isDigit(char) {
			currNumStr.WriteRune(char)
		} else {
			if currNumStr.Len() > 0 {
				num, err := strconv.ParseFloat(currNumStr.String(), 64)
				if err != nil {
					return 0, err
				}
				nums = append(nums, num)
				currNumStr.Reset()
			}
			if char == '(' {
				ops = append(ops, char)
			} else if char == ')' {
				for len(ops) > 0 && ops[len(ops)-1] != '(' {
					if len(nums) < 2 {
						return 0, errors.New("invalid expression: unmatched parentheses")
					}
					nums, ops, err = applyOperation(nums, ops)
					if err != nil {
						return 0, errors.New("invalid expression")
					}
				}
				if len(ops) == 0 {
					return 0, errors.New("invalid expression: unmatched parentheses")
				}
				ops = ops[:len(ops)-1]
			} else if isOperation(char) {
				for len(ops) > 0 && precedence(ops[len(ops)-1]) >= precedence(char) {
					if len(nums) < 2 {
						return 0, errors.New("invalid expression")
					}
					nums, ops, err = applyOperation(nums, ops)
					if err != nil {
						return 0, errors.New("invalid expression")
					}
				}
				ops = append(ops, char)
			} else {
				return 0, errors.New("invalid expression: unknown simbol")
			}
		}
	}

	if currNumStr.Len() > 0 {
		num, err := strconv.ParseFloat(currNumStr.String(), 64)
		if err != nil {
			return 0, err
		}
		nums = append(nums, num)
	}

	for len(ops) > 0 {
		if ops[len(ops)-1] == '(' {
			return 0, errors.New("invalid expression: unmatched parentheses")
		}
		if len(nums) < 2 && len(ops) > 0 {
			return 0, errors.New("incorrect")
		}
		nums, ops, err = applyOperation(nums, ops)
		if err != nil {
			return 0, errors.New("invalid expression")
		}
	}

	if len(nums) != 1 {
		return 0, errors.New("invalid expression")
	}
	return nums[0], nil
}

func isOperation(r rune) bool {
	return r == '+' || r == '-' || r == '*' || r == '/'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9' || r == '.'
}

func precedence(op rune) int {
	switch op {
	case '+', '-':
		return 1
	case '*', '/':
		return 2
	}
	return 0
}
func applyOperation(nums []float64, ops []rune) ([]float64, []rune, error) {
	if len(nums) < 2 || len(ops) == 0 {
		return nums, ops, nil
	}

	b := nums[len(nums)-1]
	a := nums[len(nums)-2]
	operator := ops[len(ops)-1]

	nums = nums[:len(nums)-2]
	ops = ops[:len(ops)-1]

	var result float64

	switch operator {
	case '+':
		result = a + b
	case '-':
		result = a - b
	case '*':
		result = a * b
	case '/':
		if b == 0 {
			return nums, ops, errors.New("invalid expression: devision by 0")
		}
		result = a / b
	}

	return append(nums, result), ops, nil
}
