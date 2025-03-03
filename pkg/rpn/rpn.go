package rpn

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/katierevinska/calculatorService/internal"
)

func Calc(expression string) (string, error) {
	var nums []string
	var ops []rune
	var currNumStr strings.Builder

	var err error
	for _, char := range expression {
		if isDigit(char) {
			currNumStr.WriteRune(char)
		} else {
			if currNumStr.Len() > 0 {
				nums = append(nums, currNumStr.String())
				currNumStr.Reset()
			}
			if char == '(' {
				ops = append(ops, char)
			} else if char == ')' {
				for len(ops) > 0 && ops[len(ops)-1] != '(' {
					if len(nums) < 2 {
						return "", errors.New("invalid expression: unmatched parentheses")
					}
					nums, ops, err = applyOperation(nums, ops)
					if err != nil {
						return "", errors.New("invalid expression")
					}
				}
				if len(ops) == 0 {
					return "", errors.New("invalid expression: unmatched parentheses")
				}
				ops = ops[:len(ops)-1]
			} else if isOperation(char) {
				for len(ops) > 0 && precedence(ops[len(ops)-1]) >= precedence(char) {
					if len(nums) < 2 {
						return "", errors.New("invalid expression")
					}
					nums, ops, err = applyOperation(nums, ops)
					if err != nil {
						return "", errors.New("invalid expression")
					}
				}
				ops = append(ops, char)
			} else {
				return "", errors.New("invalid expression: unknown simbol")
			}
		}
	}

	if currNumStr.Len() > 0 {
		nums = append(nums, currNumStr.String())
	}

	for len(ops) > 0 {
		if ops[len(ops)-1] == '(' {
			return "", errors.New("invalid expression: unmatched parentheses")
		}
		if len(nums) < 2 && len(ops) > 0 {
			return "", errors.New("incorrect")
		}
		nums, ops, err = applyOperation(nums, ops)
		if err != nil {
			return "", errors.New("invalid expression")
		}
	}

	if len(nums) != 1 {
		return "", errors.New("invalid expression")
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
func applyOperation(nums []string, ops []rune) ([]string, []rune, error) {
	if len(nums) < 2 || len(ops) == 0 {
		return nums, ops, nil
	}

	b := nums[len(nums)-1]
	a := nums[len(nums)-2]
	operator := ops[len(ops)-1]

	nums = nums[:len(nums)-2]
	ops = ops[:len(ops)-1]

	var opTime string

	switch operator {
	case '+':
		opTime = os.Getenv("TIME_ADDITION_MS")
	case '-':
		opTime = os.Getenv("TIME_SUBTRACTION_MS")
	case '*':
		opTime = os.Getenv("TIME_MULTIPLICATIONS_MS")
	case '/':
		bNum, _ := strconv.ParseFloat(b, 64)
		if 0.0 == bNum {
			return nums, ops, errors.New("invalid expression: devision by 0")
		}
		opTime = os.Getenv("TIME_DIVISIONS_MS")
	}
	store := &internal.TaskStore{}
	idResult := "id" + strconv.Itoa(int(store.GetCounter()))

	task := internal.Task{Id: idResult, Arg1: a, Arg2: b, Operation: string(operator), Operation_time: opTime}

	store.AddTask(task)
	return append(nums, idResult), ops, nil
}
