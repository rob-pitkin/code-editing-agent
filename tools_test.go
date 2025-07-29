package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReadLines(t *testing.T) {
	// test reading 1 line
	readLinesInput := json.RawMessage(`{
		"path": "test.txt",
		"start_line": 1,
		"end_line": 2
	}`)
	result, err := ReadLines(readLinesInput)
	if err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	var lines []string
	err = json.Unmarshal([]byte(result), &lines)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected := []string{"<line-1>test1</line-1>"}

	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}

	// test reading 0 lines
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 2,
		"end_line": 2
	}`)
	result, err = ReadLines(readLinesInput)
	if err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	err = json.Unmarshal([]byte(result), &lines)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected = []string{}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}

	// test out of bounds
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 3,
		"end_line": 2
	}`)
	_, err = ReadLines(readLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test reading more lines than the file has
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 1,
		"end_line": 5
	}`)
	_, err = ReadLines(readLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test reading the whole file
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 1,
		"end_line": 4
	}`)
	result, err = ReadLines(readLinesInput)
	if err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	err = json.Unmarshal([]byte(result), &lines)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected = []string{"<line-1>test1</line-1>", "<line-2>test2</line-2>", "<line-3>test3</line-3>"}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}

	// test file doesn't exist
	readLinesInput = json.RawMessage(`{
		"path":      "test2.txt",
		"start_line": 1,
		"end_line":   2
	}`)
	_, err = ReadLines(readLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
