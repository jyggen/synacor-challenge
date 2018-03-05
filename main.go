package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

func main() {
	registers := make([]uint16, 8)
	stack := newStack()
	program := readBinary("challenge.bin")
	length := uint16(len(program))
	index := uint16(0)
	debug := false
	input := ""

VM:
	for {
		var a uint16
		var b uint16
		var c uint16

		if index+1 < length {
			a = program[index+1]
		}

		if index+1 < length {
			b = program[index+2]
		}

		if index+1 < length {
			c = program[index+3]
		}

		if debug {
			fmt.Printf(
				"%v: %v %v %v %v (%v %v %v)\n",
				index,
				program[index],
				a,
				b,
				c,
				getValue(registers, a),
				getValue(registers, b),
				getValue(registers, c),
			)
		}

		switch program[index] {
		case 0: // halt: stop execution and terminate the program
			break VM
		case 1: // set: set register <a> to the value of <b>
			setRegister(registers, a, b)
			index += 2
			break
		case 2: // push: push <a> onto the stack
			stack.Push(getValue(registers, a))
			index++
			break
		case 3: // pop: remove the top element from the stack and write it into <a>; empty stack = error
			value, err := stack.Pop()

			if err != nil {
				panic(err)
			}

			setRegister(registers, a, value)
			index++
			break
		case 4: // eq: set <a> to 1 if <b> is equal to <c>; set it to 0 otherwise
			if getValue(registers, b) == getValue(registers, c) {
				setRegister(registers, a, 1)
			} else {
				setRegister(registers, a, 0)
			}

			index += 3
			break
		case 5: // gt: set <a> to 1 if <b> is greater than <c>; set it to 0 otherwise
			if getValue(registers, b) > getValue(registers, c) {
				setRegister(registers, a, 1)
			} else {
				setRegister(registers, a, 0)
			}

			index += 3
			break
		case 6: // jmp: jump to <a>
			index = getValue(registers, a) - 1
			break
		case 7: // jt: if <a> is nonzero, jump to <b>
			if getValue(registers, a) != 0 {
				index = getValue(registers, b) - 1
				break
			}

			index += 2
			break
		case 8: // jf: if <a> is zero, jump to <b>
			if getValue(registers, a) == 0 {
				index = getValue(registers, b) - 1
				break
			}

			index += 2
			break
		case 9: // add: assign into <a> the sum of <b> and <c> (modulo 32768)
			sum := (getValue(registers, b) + getValue(registers, c)) % 32768
			setRegister(registers, a, sum)
			index += 3
			break
		case 10: // mult: store into <a> the product of <b> and <c> (modulo 32768)
			value := (getValue(registers, b) * getValue(registers, c)) % 32768
			setRegister(registers, a, value)
			index += 3
			break
		case 11: // mod: store into <a> the remainder of <b> divided by <c>
			value := getValue(registers, b) % getValue(registers, c)
			setRegister(registers, a, value)
			index += 3
			break
		case 12: // and: stores into <a> the bitwise and of <b> and <c>
			value := (getValue(registers, b) & getValue(registers, c)) % 32768
			setRegister(registers, a, value)
			index += 3
			break
		case 13: // or: stores into <a> the bitwise or of <b> and <c>
			value := (getValue(registers, b) | getValue(registers, c)) % 32768
			setRegister(registers, a, value)
			index += 3
			break
		case 14: // not: stores 15-bit bitwise inverse of <b> in <a>
			setRegister(registers, a, ^getValue(registers, b)%32768)
			index += 2
			break
		case 15: // rmem: read memory at address <b> and write it to <a>
			value := getValue(registers, program[getValue(registers, b)])
			setRegister(registers, a, value)
			index += 2
			break
		case 16: // wmem: write the value from <b> into memory at address <a>
			address := getValue(registers, a)
			program[address] = getValue(registers, b)
			index += 2
			break
		case 17: // call: write the address of the next instruction to the stack and jump to <a>
			stack.Push(index + 2)
			index = getValue(registers, a) - 1
			break
		case 18: // ret: remove the top element from the stack and jump to it; empty stack = halt
			value, err := stack.Pop()

			if err != nil {
				break VM
			}

			index = getValue(registers, value) - 1
			break
		case 19: // out: write the character represented by ascii code <a> to the terminal
			fmt.Printf("%c", getValue(registers, a))
			index++
			break
		case 20: // in: read a character from the terminal and write its ascii code to <a>
			if input == "" {
				var err error

				print(":")

				reader := bufio.NewReader(os.Stdin)
				input, err = reader.ReadString('\n')

				if err != nil {
					panic(err)
				}
			}

			value := uint16(input[:1][0])
			input = input[1:]
			//debug = true

			setRegister(registers, a, getValue(registers, value))
			index++
			break
		case 21: // noop: no operation
			break
		default:
			panic(fmt.Sprintf("unknown opcode \"%v\"", program[index]))
		}

		index++
	}
}

func getValue(registers []uint16, value uint16) uint16 {
	if value < 32768 {
		return value
	}

	if value > 32775 {
		panic("overflow")
	}

	return registers[value-32768]
}

func setRegister(registers []uint16, register uint16, value uint16) {
	registers[register-32768] = getValue(registers, value)
}

func readBinary(path string) []uint16 {
	file, err := os.Open(path)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	program := make([]uint16, 0)

	for {
		var value uint16

		err = binary.Read(file, binary.LittleEndian, &value)

		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		program = append(program, value)
	}

	return program
}

type stack struct {
	lock sync.Mutex // you don't have to do this if you don't want thread safety
	s    []uint16
}

func newStack() *stack {
	return &stack{sync.Mutex{}, make([]uint16, 0)}
}

func (s *stack) Push(v uint16) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, v)
}

func (s *stack) Pop() (uint16, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := len(s.s)
	if l == 0 {
		return 0, errors.New("empty stack")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}
