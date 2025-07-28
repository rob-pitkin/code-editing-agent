package main

import (
	"encoding/json"
	"os"
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

func resetTestFile() {
	os.WriteFile("test_edit.txt", []byte("test line one\ntest line two\ntest line three\ntest line four\ntest line five"), 0644)
}

func TestEditFile(t *testing.T) {
	defer func() {
		os.Remove("test_edit.txt")
	}()

	// init file with 5 lines
	os.WriteFile("test_edit.txt", []byte("test line one\ntest line two\ntest line three\ntest line four\ntest line five"), 0644)

	editFileReplaceLineInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 1,
				"new_content": ["test line two"]
			}
		]
	}`)

	expectedReplaceLineResult := "test line two\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err := EditFile(editFileReplaceLineInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err := ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedReplaceLineResult {
		t.Fatalf("expected %v, got %v", expectedReplaceLineResult, readFileResult)
	}

	resetTestFile()

	editFileInsertLineBeforeInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "insert_line_before",
				"line_number": 1,
				"new_content": ["test line zero"]
			}
		]
	}`)

	expectedInsertLineBeforeResult := "test line zero\ntest line one\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileInsertLineBeforeInput)

	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err = ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedInsertLineBeforeResult {
		t.Fatalf("expected %v, got %v", expectedInsertLineBeforeResult, readFileResult)
	}

	resetTestFile()

	editFileInsertLineAfterInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "insert_line_after",
				"line_number": 1,
				"new_content": ["test line zero"]
			}
		]
	}`)

	expectedInsertLineAfterResult := "test line one\ntest line zero\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileInsertLineAfterInput)

	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err = ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedInsertLineAfterResult {
		t.Fatalf("expected %v, got %v", expectedInsertLineAfterResult, readFileResult)
	}

	resetTestFile()

	editFileDeleteLineInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "delete_line",
				"line_number": 1
			}
		]
	}`)

	expectedDeleteLineResult := "test line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileDeleteLineInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err = ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedDeleteLineResult {
		t.Fatalf("expected %v, got %v", expectedDeleteLineResult, readFileResult)
	}

	resetTestFile()

	editFileReplaceStringInLineInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_string_in_line",
				"line_number": 1,
				"old_string": "one",
				"new_string": "zero",
				"count": -1
			}
		]
	}`)

	expectedReplaceStringInLineResult := "test line zero\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileReplaceStringInLineInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err = ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedReplaceStringInLineResult {
		t.Fatalf("expected %v, got %v", expectedReplaceStringInLineResult, readFileResult)
	}

	resetTestFile()

	editFileAppendToFileInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "append_to_file",
				"line_number": 1,
				"new_content": ["test line six"]
			}
		]
	}`)

	expectedAppendToFileResult := "test line one\ntest line two\ntest line three\ntest line four\ntest line five\ntest line six"

	_, err = EditFile(editFileAppendToFileInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err = ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedAppendToFileResult {
		t.Fatalf("expected %v, got %v", expectedAppendToFileResult, readFileResult)
	}

	resetTestFile()

	editFileMultipleEditsInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 1,
				"new_content": ["test line zero"]
			},
			{
				"operation_type": "insert_line_before",
				"line_number": 1,
				"new_content": ["test line one"]
			},
			{
				"operation_type": "insert_line_after",
				"line_number": 5,
				"new_content": ["test line four point five"]
			},
			{
				"operation_type": "delete_line",
				"line_number": 7
			},
			{
				"operation_type": "replace_string_in_line",
				"line_number": 5,
				"old_string": "test line four",
				"new_string": "test line three point five",
				"count": -1
			},
			{
				"operation_type": "append_to_file",
				"new_content": ["test line seven"]
			}
		]
	}`)

	expectedMultipleEditsResult := "test line one\ntest line zero\ntest line two\ntest line three\ntest line three point five\ntest line four point five\ntest line seven"

	_, err = EditFile(editFileMultipleEditsInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	readFileResult, err = ReadFile(json.RawMessage(`{"path": "test_edit.txt"}`))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if readFileResult != expectedMultipleEditsResult {
		t.Fatalf("expected %v, got %v", expectedMultipleEditsResult, readFileResult)
	}

	editFileInvalidInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line"
			}
		]
	}`)
	_, err = EditFile(editFileInvalidInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	editFileInvalidOperationTypeInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "invalid_operation_type"
			}
		]
	}`)
	_, err = EditFile(editFileInvalidOperationTypeInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	editFileInvalidLineNumberInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 0,
				"new_content": ["test line zero"]
			}
		]
	}`)
	_, err = EditFile(editFileInvalidLineNumberInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
