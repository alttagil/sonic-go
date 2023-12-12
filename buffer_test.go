// Copyright (c) 2023 Alexander Khudich
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sonic

import (
	"io"
	"reflect"
	"testing"
)

func TestBuffer_Write(t *testing.T) {
	b := &Buffer[int]{} // Assuming int for simplicity. You can change the type as needed.
	values := []int{1, 2, 3, 4, 5}

	for _, v := range values {
		err := b.Write(v)
		if err != nil {
			t.Errorf("Error writing value %v to buffer: %v", v, err)
		}
	}

	expected := values
	actual := b.buf

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Write: Expected %v, but got %v", expected, actual)
	}
}

func TestBuffer_WriteSlice(t *testing.T) {
	b := &Buffer[int]{} // Assuming int for simplicity. You can change the type as needed.
	slice := []int{1, 2, 3, 4, 5}

	err := b.WriteSlice(slice)
	if err != nil {
		t.Errorf("Error writing slice to buffer: %v", err)
	}

	expected := slice
	actual := b.buf

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("WriteSlice: Expected %v, but got %v", expected, actual)
	}
}

func TestBuffer_Read(t *testing.T) {
	b := &Buffer[int]{} // Assuming int for simplicity. You can change the type as needed.
	values := []int{1, 2, 3, 4, 5}

	b.buf = values

	for _, expected := range values {
		actual, err := b.Read()
		if err != nil {
			t.Errorf("Error reading from buffer: %v", err)
		}

		if actual != expected {
			t.Errorf("Read: Expected %v, but got %v", expected, actual)
		}
	}

	// Reading beyond the available values should return EOF
	_, err := b.Read()
	if err != io.EOF {
		t.Errorf("Expected EOF after reading all values, but got: %v", err)
	}
}

func TestBuffer_ReadSlice(t *testing.T) {
	b := &Buffer[int]{} // Assuming int for simplicity. You can change the type as needed.
	slice := []int{1, 2, 3, 4, 5}

	b.buf = slice

	expected := slice
	actual, err := b.ReadSlice(len(slice))
	if err != nil {
		t.Errorf("Error reading slice from buffer: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("ReadSlice: Expected %v, but got %v", expected, actual)
	}

	// Reading beyond the available values should return EOF
	_, err = b.ReadSlice(1)
	if err != io.EOF {
		t.Errorf("Expected EOF after reading all values, but got: %v", err)
	}
}
